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

func New(protector protector.Protector) *Server {
	s := &Server{
		protector: protector,
	}
	s.router = chi.NewRouter()
	s.router.Use(middleware.Logger)
	s.router.Route("/v1", func(r chi.Router) {
		r.Route("/{network}", func(r chi.Router) {
			r.Use(networkCtx)
			r.Route("/slashable", func(r chi.Router) {
				r.Post("/proposal", s.slashableProposal)
				r.Post("/attestation", s.slashableAttestation)
			})
		})
	})
	return s
}

func (s *Server) slashableProposal(w http.ResponseWriter, r *http.Request) {
	type slashableProposalRequest struct {
		PubKey      jsonPubKey          `json:"pub_key"`
		SigningRoot jsonRoot            `json:"signing_root"`
		Block       *altair.BeaconBlock `json:"block"`
	}
	type slashableProposalResponse struct {
		Slashable bool   `json:"slashable"`
		Reason    string `json:"slashing"`
	}

	var request slashableProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := s.protector.CheckProposal(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Block,
	)
	response := slashableProposalResponse{}
	if err != nil {
		response.Slashable = true
		response.Reason = err.Error()
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) slashableAttestation(w http.ResponseWriter, r *http.Request) {
	type slashableAttestationRequest struct {
		PubKey      jsonPubKey          `json:"pub_key"`
		SigningRoot jsonRoot            `json:"signing_root"`
		Attestation *phase0.Attestation `json:"attestation"`
	}
	type slashableAttestationResponse struct {
		Slashable bool   `json:"slashable"`
		Reason    string `json:"slashing"`
	}

	var request slashableAttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := s.protector.CheckAttestation(
		r.Context(),
		getNetwork(r.Context()),
		phase0.BLSPubKey(request.PubKey),
		phase0.Root(request.SigningRoot),
		request.Attestation,
	)
	response := slashableAttestationResponse{}
	if err != nil {
		response.Slashable = true
		response.Reason = err.Error()
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
