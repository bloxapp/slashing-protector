package http

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	types "github.com/prysmaticlabs/eth2-types"
	"go.uber.org/zap"
)

type Server struct {
	logger    *zap.Logger
	protector protector.Protector
	router    *chi.Mux
}

func NewServer(logger *zap.Logger, protector protector.Protector) *Server {
	s := &Server{
		logger:    logger,
		protector: protector,
	}
	s.router = chi.NewRouter()
	s.router.Use(middleware.Logger)
	s.router.Use(render.SetContentType(render.ContentTypeJSON))
	s.router.Mount("/debug", middleware.Profiler())
	s.router.Route("/v1", func(r chi.Router) {
		r.Route("/{network}", func(r chi.Router) {
			r.Use(networkCtx)
			r.Route("/slashable", func(r chi.Router) {
				r.Post("/proposal", s.handleCheckProposal)
				r.Post("/attestation", s.handleCheckAttestation)
			})
			r.Get("/history/{pub_key}", s.handleHistory)
		})
	})
	return s
}

type checkProposalRequest struct {
	PubKey      jsonPubKey  `json:"pub_key"`
	SigningRoot jsonRoot    `json:"signing_root"`
	Slot        phase0.Slot `json:"block"`
}

func (s *Server) handleCheckProposal(w http.ResponseWriter, r *http.Request) {
	var request checkProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		render.JSON(w, r, &checkResponse{
			StatusCode: http.StatusBadRequest,
			Error:      err.Error(),
		})
		return
	}

	var resp checkResponse
	var err error
	resp.Check, err = s.protector.CheckProposal(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Slot,
	)
	if err != nil {
		resp.StatusCode = http.StatusInternalServerError
		resp.Error = err.Error()
	}
	render.JSON(w, r, resp)
}

type checkAttestationRequest struct {
	PubKey      jsonPubKey              `json:"pub_key"`
	SigningRoot jsonRoot                `json:"signing_root"`
	Data        *phase0.AttestationData `json:"attestation"`
}

func (s *Server) handleCheckAttestation(w http.ResponseWriter, r *http.Request) {
	var request checkAttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("failed to decode checkAttestationRequest", zap.Error(err))
		render.JSON(w, r, &checkResponse{
			StatusCode: http.StatusBadRequest,
			Error:      err.Error(),
		})
		return
	}

	var resp checkResponse
	var err error
	resp.Check, err = s.protector.CheckAttestation(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Data,
	)
	if err != nil {
		s.logger.Error(
			"failed at CheckAttestation",
			zap.Any("attestation", request),
			zap.Error(err),
		)
		resp.StatusCode = http.StatusInternalServerError
		resp.Error = err.Error()
	}
	render.JSON(w, r, resp)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	// Decode the public key.
	var pubKey phase0.BLSPubKey
	b, err := hex.DecodeString(strings.TrimPrefix(chi.URLParam(r, "pub_key"), "0x"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	copy(pubKey[:], b)

	// Get the history.
	history, err := s.protector.History(r.Context(), getNetwork(r.Context()), pubKey)
	if err != nil {
		s.logger.Error("failed to get history", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Compact the proposals & attestations for a smaller JSON response.
	type proposal struct {
		SigningRoot string     `json:"signing_root"`
		Slot        types.Slot `json:"slot"`
	}
	proposals := make([]proposal, len(history.Proposals))
	for i, p := range history.Proposals {
		proposals[i] = proposal{
			SigningRoot: hex.EncodeToString(p.SigningRoot[:]),
			Slot:        p.Slot,
		}
	}

	type attestation struct {
		SigningRoot string      `json:"signing_root"`
		Source      types.Epoch `json:"source"`
		Target      types.Epoch `json:"target"`
	}
	attestations := make([]attestation, len(history.Attestations))
	for i, a := range history.Attestations {
		attestations[i] = attestation{
			SigningRoot: hex.EncodeToString(a.SigningRoot[:]),
			Source:      a.Source,
			Target:      a.Target,
		}
	}

	// Respond with the history.
	render.JSON(w, r, struct {
		Proposals    []proposal    `json:"proposals"`
		Attestations []attestation `json:"attestations"`
	}{
		Proposals:    proposals,
		Attestations: attestations,
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func networkCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		if network == "" {
			http.Error(w, "network parameter is required", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), "network", network)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getNetwork(ctx context.Context) string {
	return ctx.Value("network").(string)
}
