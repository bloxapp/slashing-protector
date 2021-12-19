package http

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router    *chi.Mux
	protector protector.Protector
}

func NewServer(protector protector.Protector) *Server {
	s := &Server{
		protector: protector,
	}
	s.router = chi.NewRouter()
	s.router.Use(middleware.Logger)
	s.router.Route("/v1", func(r chi.Router) {
		r.Route("/{network}", func(r chi.Router) {
			r.Use(networkCtx)
			r.Route("/slashable", func(r chi.Router) {
				r.Post("/proposal", s.handleCheckProposal)
				r.Post("/attestation", s.handleCheckAttestation)
			})
		})
	})
	return s
}

type checkProposalRequest struct {
	PubKey      jsonPubKey          `json:"pub_key"`
	SigningRoot jsonRoot            `json:"signing_root"`
	Block       *altair.BeaconBlock `json:"block"`
}

func (s *Server) handleCheckProposal(w http.ResponseWriter, r *http.Request) {
	var request checkProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	check, err := s.protector.CheckProposal(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Block,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(check); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type checkAttestationRequest struct {
	PubKey      jsonPubKey              `json:"pub_key"`
	SigningRoot jsonRoot                `json:"signing_root"`
	Data        *phase0.AttestationData `json:"attestation"`
}

func (s *Server) handleCheckAttestation(w http.ResponseWriter, r *http.Request) {
	var request checkAttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	check, err := s.protector.CheckAttestation(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Data,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Check *protector.Check
		Req   checkAttestationRequest
	}{check, request}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

type jsonPubKey phase0.BLSPubKey

func (j *jsonPubKey) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return err
	}
	copy(j[:], v)
	return nil
}

type jsonRoot phase0.Root

func (j *jsonRoot) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return err
	}
	copy(j[:], v)
	return nil
}
