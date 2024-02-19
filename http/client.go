package http

import (
	"context"
	"net/http"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/carlmjohnson/requests"
	"github.com/pkg/errors"

	"github.com/bloxapp/slashing-protector/protector"
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
	var resp checkResponse
	err := requests.
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
	if resp.Timestamp != req.Timestamp {
		return nil, errors.New("timestamp mismatch")
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
		Slot:        compatibleSlot(slot),
	}
	var resp checkResponse
	err := requests.
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
	if resp.Timestamp != req.Timestamp {
		return nil, errors.New("timestamp mismatch")
	}
	return resp.Check, nil
}
