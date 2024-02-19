package http

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/cespare/xxhash/v2"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
)

type requestHasher interface {
	Hash() (uint64, error)
}

type checkProposalRequest struct {
	Timestamp   int64       `json:"timestamp"`
	PubKey      jsonPubKey  `json:"pub_key"`
	SigningRoot jsonRoot    `json:"signing_root"`
	Slot        phase0.Slot `json:"block"`
}

func (r *checkProposalRequest) Hash() (uint64, error) {
	h := xxhash.New()
	writeUint64(h, uint64(r.Timestamp))
	h.Write(r.PubKey[:])
	h.Write(r.SigningRoot[:])
	writeUint64(h, uint64(r.Slot))
	return h.Sum64(), nil
}

type checkAttestationRequest struct {
	Timestamp   int64                  `json:"timestamp"`
	PubKey      jsonPubKey             `json:"pub_key"`
	SigningRoot jsonRoot               `json:"signing_root"`
	Data        phase0.AttestationData `json:"attestation"`
}

func (r *checkAttestationRequest) Hash() (uint64, error) {
	if r.Data.Source == nil || r.Data.Target == nil {
		return 0, errors.New("source and target are required")
	}
	h := xxhash.New()
	writeUint64(h, uint64(r.Timestamp))
	h.Write(r.PubKey[:])
	h.Write(r.SigningRoot[:])
	writeUint64(h, uint64(r.Data.Slot))
	writeUint64(h, uint64(r.Data.Index))
	h.Write(r.Data.BeaconBlockRoot[:])
	h.Write(r.Data.Source.Root[:])
	writeUint64(h, uint64(r.Data.Source.Epoch))
	h.Write(r.Data.Target.Root[:])
	writeUint64(h, uint64(r.Data.Target.Epoch))
	return h.Sum64(), nil
}

func writeUint64(h *xxhash.Digest, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	h.Write(buf[:])
}

type checkResponse struct {
	Hash       uint64           `json:"hash"`
	Check      *protector.Check `json:"check"`
	StatusCode int              `json:"status_code"`
	Error      string           `json:"error,omitempty"`
}

func (c *checkResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if c.StatusCode != 0 {
		render.Status(r, c.StatusCode)
	}
	render.JSON(w, r, c)
	return nil
}

type jsonPubKey phase0.BLSPubKey

func (j jsonPubKey) MarshalJSON() ([]byte, error) {
	return []byte(`"0x` + hex.EncodeToString(j[:]) + `"`), nil
}

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

func (j jsonRoot) MarshalJSON() ([]byte, error) {
	return []byte(`"0x` + hex.EncodeToString(j[:]) + `"`), nil
}

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
