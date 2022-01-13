package protector

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector/kvpool"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/slashings"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// Check is the result of an attestation check or a proposal check.
type Check struct {
	Slashable bool   `json:"slashable"`
	Reason    string `json:"slashing,omitempty"`
}

// slashable returns a slashable Check with the given reason.
func slashable(reason string, args ...interface{}) *Check {
	return &Check{
		Slashable: true,
		Reason:    fmt.Sprintf(reason, args...),
	}
}

// Protector is the interface for the slashing protection.
type Protector interface {
	// CheckAttestation an attestation for a potential slashing.
	CheckAttestation(
		ctx context.Context,
		network string,
		pubKey phase0.BLSPubKey,
		signingRoot phase0.Root,
		attestation *phase0.AttestationData,
	) (*Check, error)

	// CheckProposal checks a proposal for a potential slashing.
	CheckProposal(
		ctx context.Context,
		network string,
		pubKey phase0.BLSPubKey,
		signingRoot phase0.Root,
		slot phase0.Slot,
	) (*Check, error)
}

type ProtectorCloser interface {
	Protector

	// Close closes the database.
	Close() error
}

type protector struct {
	pool *kvpool.Pool
}

func New(dir string) ProtectorCloser {
	return &protector{
		pool: kvpool.New(dir),
	}
}

// Close closes the database.
func (p *protector) Close() error {
	return p.pool.Close()
}

func (p *protector) CheckAttestation(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	data *phase0.AttestationData,
) (*Check, error) {
	conn, err := p.pool.Acquire(ctx, network, pubKey)
	if err != nil {
		return nil, errors.Wrap(err, "kvpool.Acquire")
	}
	defer conn.Release() // TODO: log the error

	// Based on EIP3076, validator should refuse to sign any attestation with source epoch less
	// than the minimum source epoch present in that signer’s attestations.
	lowestSourceEpoch, exists, err := conn.LowestSignedSourceEpoch(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	if exists && types.Epoch(data.Source.Epoch) < lowestSourceEpoch {
		return slashable(
			"could not sign attestation lower than lowest source epoch in db, %d < %d",
			data.Source.Epoch,
			lowestSourceEpoch,
		), nil
	}
	existingSigningRoot, err := conn.SigningRootAtTargetEpoch(
		ctx,
		pubKey,
		types.Epoch(data.Target.Epoch),
	)
	if err != nil {
		return nil, err
	}
	signingRootsDiffer := slashings.SigningRootsDiffer(existingSigningRoot, signingRoot)

	// Based on EIP3076, validator should refuse to sign any attestation with target epoch less
	// than or equal to the minimum target epoch present in that signer’s attestations.
	lowestTargetEpoch, exists, err := conn.LowestSignedTargetEpoch(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	if signingRootsDiffer && exists && types.Epoch(data.Target.Epoch) <= lowestTargetEpoch {
		return slashable(
			"could not sign attestation lower than or equal to lowest target epoch in db, %d <= %d",
			data.Target.Epoch,
			lowestTargetEpoch,
		), nil
	}

	// Convert the attestation to a type compatible with Prysm's kv.
	prysmAtt := &ethpb.IndexedAttestation{
		// TODO: AttestingIndices and Signatures are currently not used in
		// Prysm's attestation check, but this might change and break the
		// CheckSlashableAttestation call.
		AttestingIndices: []uint64{},
		Signature:        nil,

		Data: &ethpb.AttestationData{
			Slot:            types.Slot(data.Slot),
			CommitteeIndex:  types.CommitteeIndex(data.Index),
			BeaconBlockRoot: data.BeaconBlockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: types.Epoch(data.Source.Epoch),
				Root:  data.Source.Root[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: types.Epoch(data.Target.Epoch),
				Root:  data.Target.Root[:],
			},
		},
	}
	slashingKind, err := conn.CheckSlashableAttestation(ctx, pubKey, signingRoot, prysmAtt)
	if err != nil {
		switch slashingKind {
		case kv.DoubleVote:
			return slashable("Attestation is slashable as it is a double vote: %v", err), nil
		case kv.SurroundingVote:
			return slashable(
				"Attestation is slashable as it is surrounding a previous attestation: %v",
				err,
			), nil
		case kv.SurroundedVote:
			return slashable(
				"Attestation is slashable as it is surrounded by a previous attestation: %v",
				err,
			), nil
		}
		return nil, err
	}
	if err := conn.SaveAttestationForPubKey(ctx, pubKey, signingRoot, prysmAtt); err != nil {
		return nil, errors.Wrap(err, "could not save attestation history for validator public key")
	}
	return &Check{}, nil
}

func (p *protector) CheckProposal(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	slot phase0.Slot,
) (*Check, error) {
	conn, err := p.pool.Acquire(ctx, network, pubKey)
	if err != nil {
		return nil, errors.Wrap(err, "kvpool.Acquire")
	}
	defer conn.Release() // TODO: log the error

	prevSigningRoot, proposalAtSlotExists, err := conn.ProposalHistoryForSlot(
		ctx,
		pubKey,
		types.Slot(slot),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get proposal history")
	}

	lowestSignedProposalSlot, lowestProposalExists, err := conn.LowestSignedProposal(ctx, pubKey)
	if err != nil {
		return nil, err
	}

	// If a proposal exists in our history for the slot, we check the following:
	// If the signing root is empty (zero hash), then we consider it slashable. If signing root is not empty,
	// we check if it is different than the incoming block's signing root. If that is the case,
	// we consider that proposal slashable.
	signingRootIsDifferent := prevSigningRoot == params.BeaconConfig().ZeroHash ||
		prevSigningRoot != signingRoot
	if proposalAtSlotExists && signingRootIsDifferent {
		return slashable(
			"attempted to sign a double proposal, block rejected by local protection",
		), nil
	}

	// Based on EIP3076, validator should refuse to sign any proposal with slot less
	// than or equal to the minimum signed proposal present in the DB for that public key.
	// In the case the slot of the incoming block is equal to the minimum signed proposal, we
	// then also check the signing root is different.
	if lowestProposalExists && signingRootIsDifferent &&
		lowestSignedProposalSlot >= types.Slot(slot) {
		return slashable(
			"could not sign block with slot <= lowest signed slot in db, lowest signed slot: %d >= block slot: %d",
			lowestSignedProposalSlot,
			slot,
		), nil
	}

	if err := conn.SaveProposalHistoryForSlot(ctx, pubKey, types.Slot(slot), signingRoot[:]); err != nil {
		return nil, errors.Wrap(err, "failed to save updated proposal history")
	}
	return &Check{}, nil
}
