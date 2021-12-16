package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/pkg/errors"
)

type Client struct {
	http *http.Client
	url  *url.URL
}

func NewClient(http *http.Client, addr string) (*Client, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url.Parse")
	}
	return &Client{
		http: http,
		url:  u,
	}, nil
}

func (c *Client) CheckAttestation(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	attestation *phase0.Attestation,
) (*protector.Check, error) {
	request := &checkAttestationRequest{
		PubKey:      jsonPubKey(pubKey),
		SigningRoot: jsonRoot(signingRoot),
		Attestation: attestation,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post(
		c.url.ResolveReference(&url.URL{Path: fmt.Sprintf("/v1/%s/attestation", network)}).String(),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, errors.Wrap(err, "http.Post")
	}
	defer resp.Body.Close()

	var check protector.Check
	if err := json.NewDecoder(resp.Body).Decode(&check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (c *Client) CheckProposal(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	block *altair.BeaconBlock,
) (*protector.Check, error) {
	request := &checkProposalRequest{
		PubKey:      jsonPubKey(pubKey),
		SigningRoot: jsonRoot(signingRoot),
		Block:       block,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post(
		c.url.ResolveReference(&url.URL{Path: fmt.Sprintf("/v1/%s/proposal", network)}).String(),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, errors.Wrap(err, "http.Post")
	}
	defer resp.Body.Close()

	var check protector.Check
	if err := json.NewDecoder(resp.Body).Decode(&check); err != nil {
		return nil, err
	}
	return &check, nil
}
