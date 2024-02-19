package http

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/require"
)

func TestCheckProposalRequest_Hash(t *testing.T) {
	mock := checkProposalRequest{
		Timestamp:   1000,
		PubKey:      jsonPubKey([48]byte{1, 2, 3}),
		SigningRoot: jsonRoot([32]byte{4, 5, 6}),
		Slot:        7,
	}
	hasher := newHasher(mock.Hash)

	// Expect known hash.
	hasher.expect(t, uint64(0xdc56a40de8bcb724))

	// Expect repeatable hash.
	hasher.expect(t, uint64(0xdc56a40de8bcb724))

	// Expect different hash when a field changes.
	mock.Timestamp = 1001
	hasher.expectUnique(t)
	mock.PubKey = jsonPubKey([48]byte{1, 2, 4})
	hasher.expectUnique(t)
	mock.SigningRoot = jsonRoot([32]byte{4, 5, 7})
	hasher.expectUnique(t)
	mock.Slot = 8
	hasher.expectUnique(t)
}

func TestCheckAttestationRequest_Hash(t *testing.T) {
	mock := checkAttestationRequest{
		Timestamp:   1000,
		PubKey:      jsonPubKey([48]byte{1, 2, 3}),
		SigningRoot: jsonRoot([32]byte{4, 5, 6}),
		Data: phase0.AttestationData{
			Slot:            7,
			Index:           8,
			BeaconBlockRoot: [32]byte{9, 10, 11},
			Source: &phase0.Checkpoint{
				Root:  [32]byte{12, 13, 14},
				Epoch: 15,
			},
			Target: &phase0.Checkpoint{
				Root:  [32]byte{16, 17, 18},
				Epoch: 19,
			},
		},
	}
	hasher := newHasher(mock.Hash)

	// Expect known hash.
	hasher.expect(t, uint64(0x629b4bff388aeb6a))

	// Expect repeatable hash.
	hasher.expect(t, uint64(0x629b4bff388aeb6a))

	// Expect different hash when a field changes.
	mock.Timestamp = 1001
	hasher.expectUnique(t)
	mock.PubKey = jsonPubKey([48]byte{1, 2, 4})
	hasher.expectUnique(t)
	mock.SigningRoot = jsonRoot([32]byte{4, 5, 7})
	hasher.expectUnique(t)
	mock.Data.Slot = 20
	hasher.expectUnique(t)
	mock.Data.Index = 21
	hasher.expectUnique(t)
	mock.Data.BeaconBlockRoot = [32]byte{22, 23, 24}
	hasher.expectUnique(t)
	mock.Data.Source.Root = [32]byte{25, 26, 27}
	hasher.expectUnique(t)
	mock.Data.Source.Epoch = 28
	hasher.expectUnique(t)
	mock.Data.Target.Root = [32]byte{29, 30, 31}
	hasher.expectUnique(t)
	mock.Data.Target.Epoch = 32
	hasher.expectUnique(t)
}

type hasher struct {
	fn     func() (uint64, error)
	hashes map[uint64]struct{}
	last   uint64
}

func newHasher(fn func() (uint64, error)) *hasher {
	return &hasher{
		fn:     fn,
		hashes: make(map[uint64]struct{}),
	}
}

func (h *hasher) hash(t *testing.T) uint64 {
	hash, err := h.fn()
	require.NoError(t, err)
	h.last = hash
	return hash
}

func (h *hasher) expect(t *testing.T, expected uint64) {
	hash := h.hash(t)
	require.Equal(t, expected, hash)
}

func (h *hasher) expectUnique(t *testing.T) {
	hash := h.hash(t)
	_, exists := h.hashes[hash]
	require.False(t, exists)
	h.hashes[hash] = struct{}{}
}
