package http

import (
	"context"
	"net/http"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/carlmjohnson/requests"
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
	attestation *phase0.Attestation,
) (resp *protector.Check, err error) {
	err = requests.
		URL(c.baseURL).
		Client(c.http).
		Pathf("/v1/%s/attestation", network).
		BodyJSON(&checkAttestationRequest{
			PubKey:      jsonPubKey(pubKey),
			SigningRoot: jsonRoot(signingRoot),
			Attestation: attestation,
		}).
		ToJSON(resp).
		Fetch(ctx)
	return
}

func (c *Client) CheckProposal(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	block *altair.BeaconBlock,
) (resp *protector.Check, err error) {
	err = requests.
		URL(c.baseURL).
		Client(c.http).
		Pathf("/v1/%s/proposal", network).
		BodyJSON(&checkProposalRequest{
			PubKey:      jsonPubKey(pubKey),
			SigningRoot: jsonRoot(signingRoot),
			Block:       block,
		}).
		ToJSON(resp).
		Fetch(ctx)
	return
}
