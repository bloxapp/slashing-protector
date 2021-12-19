package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
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
	s.router.Use(render.SetContentType(render.ContentTypeJSON))
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
		resp.StatusCode = http.StatusInternalServerError
		resp.Error = err.Error()
	}
	render.JSON(w, r, resp)
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
