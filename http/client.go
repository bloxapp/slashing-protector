package http

import (
	"context"
	"net/http"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/carlmjohnson/requests"
	"github.com/pkg/errors"
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient(http *http.Client, addr string) *Client {
	return &Client{
		http:    http,
		baseURL: addr,
	}
}

func (c *Client) CheckAttestation(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	data *phase0.AttestationData,
) (*protector.Check, error) {
	if data == nil {
		return nil, errors.New("data is required")
	}

	req := &checkAttestationRequest{
		Timestamp:   time.Now().UnixNano(),
		PubKey:      jsonPubKey(pubKey),
		SigningRoot: jsonRoot(signingRoot),
		Data:        *data,
	}
	hash, err := req.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash request")
	}
	if hash == 0 {
		return nil, errors.New("hash is zero")
	}

	var resp checkResponse
	err = requests.
		URL(c.baseURL).
		Client(c.http).
		Pathf("/v1/%s/slashable/attestation", network).
		BodyJSON(req).
		AddValidator(nil). // Don't check http.StatusOK
		ToJSON(&resp).
		Fetch(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch")
	}
	if resp.Error != "" {
		return nil, errors.Wrap(errors.New(resp.Error), "error from server")
	}
	if resp.Hash != hash {
		return nil, errors.New("mismatching hash")
	}
	if resp.Check == nil {
		return nil, errors.New("malformed response: check is nil")
	}
	return resp.Check, nil
}

func (c *Client) CheckProposal(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	slot phase0.Slot,
) (*protector.Check, error) {
	req := &checkProposalRequest{
		PubKey:      jsonPubKey(pubKey),
		SigningRoot: jsonRoot(signingRoot),
		Slot:        slot,
	}
	hash, err := req.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash request")
	}
	if hash == 0 {
		return nil, errors.New("hash is zero")
	}

	var resp checkResponse
	err = requests.
		URL(c.baseURL).
		Client(c.http).
		Pathf("/v1/%s/slashable/proposal", network).
		BodyJSON(req).
		AddValidator(nil). // Don't check http.StatusOK
		ToJSON(&resp).
		Fetch(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch")
	}
	if resp.Error != "" {
		return nil, errors.Wrap(errors.New(resp.Error), "error from server")
	}
	if resp.Hash != hash {
		return nil, errors.New("mismatching hash")
	}
	if resp.Check == nil {
		return nil, errors.New("malformed response: check is nil")
	}
	return resp.Check, nil
}

func (c *Client) History(ctx context.Context, network string, pubKey phase0.BLSPubKey) (history *protector.History, err error) {
	// TODO: Implement.
	return nil, nil
}
