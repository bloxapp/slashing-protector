package http

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/go-chi/render"
)

type checkResponse struct {
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
