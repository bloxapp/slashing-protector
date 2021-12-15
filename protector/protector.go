package protector

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector/kvpool"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/slashings"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

type Protector interface {
	CheckAttestation(
		ctx context.Context,
		network string,
		pubKey phase0.BLSPubKey,
		signingRoot phase0.Root,
		attestation *phase0.Attestation,
	) error

	CheckProposal(
		ctx context.Context,
		network string,
		pubKey phase0.BLSPubKey,
		signingRoot phase0.Root,
		block *altair.BeaconBlock,
	) error
}

type protector struct {
	pool *kvpool.Pool
}

func New(dir string) Protector {
	return &protector{
		pool: kvpool.New(dir),
	}
}

func (p *protector) CheckAttestation(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	att *phase0.Attestation,
) error {
	conn, err := p.pool.Acquire(ctx, network, pubKey)
	if err != nil {
		return errors.Wrap(err, "kvpool.Acquire")
	}
	defer conn.Release()

	// Based on EIP3076, validator should refuse to sign any attestation with source epoch less
	// than the minimum source epoch present in that signer’s attestations.
	lowestSourceEpoch, exists, err := conn.LowestSignedSourceEpoch(ctx, pubKey)
	if err != nil {
		return err
	}
	if exists && types.Epoch(att.Data.Source.Epoch) < lowestSourceEpoch {
		return fmt.Errorf(
			"could not sign attestation lower than lowest source epoch in db, %d < %d",
			att.Data.Source.Epoch,
			lowestSourceEpoch,
		)
	}
	existingSigningRoot, err := conn.SigningRootAtTargetEpoch(ctx, pubKey, types.Epoch(att.Data.Target.Epoch))
	if err != nil {
		return err
	}
	signingRootsDiffer := slashings.SigningRootsDiffer(existingSigningRoot, signingRoot)

	// Based on EIP3076, validator should refuse to sign any attestation with target epoch less
	// than or equal to the minimum target epoch present in that signer’s attestations.
	lowestTargetEpoch, exists, err := conn.LowestSignedTargetEpoch(ctx, pubKey)
	if err != nil {
		return err
	}
	if signingRootsDiffer && exists && types.Epoch(att.Data.Target.Epoch) <= lowestTargetEpoch {
		return fmt.Errorf(
			"could not sign attestation lower than or equal to lowest target epoch in db, %d <= %d",
			att.Data.Target.Epoch,
			lowestTargetEpoch,
		)
	}

	// Convert the attestation to a type compatible with Prysm's kv.
	prysmAtt := &ethpb.IndexedAttestation{
		// TODO: is AttestingIndices needed?
		AttestingIndices: []uint64{},

		Data: &ethpb.AttestationData{
			Slot:            types.Slot(att.Data.Slot),
			CommitteeIndex:  types.CommitteeIndex(att.Data.Index),
			BeaconBlockRoot: att.Data.BeaconBlockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: types.Epoch(att.Data.Source.Epoch),
				Root:  att.Data.Source.Root[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: types.Epoch(att.Data.Target.Epoch),
				Root:  att.Data.Target.Root[:],
			},
		},
		Signature: att.Signature[:],
	}
	slashingKind, err := conn.CheckSlashableAttestation(ctx, pubKey, signingRoot, prysmAtt)
	if err != nil {
		switch slashingKind {
		case kv.DoubleVote:
			return errors.Wrap(err, "Attestation is slashable as it is a double vote")
		case kv.SurroundingVote:
			return errors.Wrap(err, "Attestation is slashable as it is surrounding a previous attestation")
		case kv.SurroundedVote:
			return errors.Wrap(err, "Attestation is slashable as it is surrounded by a previous attestation")
		}
		return err
	}
	if err := conn.SaveAttestationForPubKey(ctx, pubKey, signingRoot, prysmAtt); err != nil {
		return errors.Wrap(err, "could not save attestation history for validator public key")
	}
	return nil
}

var (
	failedBlockSignLocalErr    = "attempted to sign a double proposal, block rejected by local protection"
	failedBlockSignExternalErr = "attempted a double proposal, block rejected by remote slashing protection"
)

func (p *protector) CheckProposal(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
	signingRoot phase0.Root,
	block *altair.BeaconBlock,
) error {
	conn, err := p.pool.Acquire(ctx, network, pubKey)
	if err != nil {
		return errors.Wrap(err, "kvpool.Acquire")
	}

	prevSigningRoot, proposalAtSlotExists, err := conn.ProposalHistoryForSlot(ctx, pubKey, types.Slot(block.Slot))
	if err != nil {
		return errors.Wrap(err, "failed to get proposal history")
	}

	lowestSignedProposalSlot, lowestProposalExists, err := conn.LowestSignedProposal(ctx, pubKey)
	if err != nil {
		return err
	}

	// If a proposal exists in our history for the slot, we check the following:
	// If the signing root is empty (zero hash), then we consider it slashable. If signing root is not empty,
	// we check if it is different than the incoming block's signing root. If that is the case,
	// we consider that proposal slashable.
	signingRootIsDifferent := prevSigningRoot == params.BeaconConfig().ZeroHash || prevSigningRoot != signingRoot
	if proposalAtSlotExists && signingRootIsDifferent {
		return errors.New(failedBlockSignLocalErr)
	}

	// Based on EIP3076, validator should refuse to sign any proposal with slot less
	// than or equal to the minimum signed proposal present in the DB for that public key.
	// In the case the slot of the incoming block is equal to the minimum signed proposal, we
	// then also check the signing root is different.
	if lowestProposalExists && signingRootIsDifferent && lowestSignedProposalSlot >= types.Slot(block.Slot) {
		return fmt.Errorf(
			"could not sign block with slot <= lowest signed slot in db, lowest signed slot: %d >= block slot: %d",
			lowestSignedProposalSlot,
			block.Slot,
		)
	}

	if err := conn.SaveProposalHistoryForSlot(ctx, pubKey, types.Slot(block.Slot), signingRoot[:]); err != nil {
		return errors.Wrap(err, "failed to save updated proposal history")
	}
	return nil
}
