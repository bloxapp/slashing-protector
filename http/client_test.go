package http

import (
	"context"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/slashing-protector/protector"
	"github.com/stretchr/testify/require"
)

func TestClient_CheckAttestation_Valid(t *testing.T) {
	client, _ := setupClient(t)

	// Check a valid attestation.
	check, err := client.CheckAttestation(context.Background(), "mainnet", phase0.BLSPubKey{}, phase0.Root{}, createAttestationData(0, 1))
	require.NoError(t, err)
	require.False(t, check.Slashable, "unexpected slashing: %s", check.Reason)

	// Same signing root, same key -> expect slashing.
	check, err = client.CheckAttestation(context.Background(), "mainnet", phase0.BLSPubKey{}, phase0.Root{0x1}, createAttestationData(0, 1))
	require.NoError(t, err)
	require.True(t, check.Slashable, "expected slashing")

	// Same signing root, different key -> no slashing.
	check, err = client.CheckAttestation(context.Background(), "mainnet", phase0.BLSPubKey{0x1}, phase0.Root{}, createAttestationData(0, 2))
	require.NoError(t, err)
	require.False(t, check.Slashable, "unexpected slashing: %s", check.Reason)

	// Same signing root, same key, next epoch -> no slashing.
	check, err = client.CheckAttestation(context.Background(), "mainnet", phase0.BLSPubKey{}, phase0.Root{}, createAttestationData(1, 2))
	require.NoError(t, err)
	require.False(t, check.Slashable, "unexpected slashing: %s", check.Reason)
}

func TestClient_CheckAttestation_Concurrent(t *testing.T) {
	client, _ := setupClient(t)

	// Spawn a bunch of workers.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Check attestation for the same public keys as the other workers,
			// hoping to do so for the same key at the same time.
			for _, j := range rand.Perm(4) {
				pubKey := phase0.BLSPubKey{byte(j)}

				epoch := phase0.Epoch(rand.Intn(5))

				_, err := client.CheckAttestation(context.Background(), "mainnet", pubKey, phase0.Root{byte(i)}, createAttestationData(epoch, epoch+1))
				require.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestClient_CheckAttestation_Offline(t *testing.T) {
	client, server := setupClient(t)
	server.Close()
	_, err := client.CheckAttestation(context.Background(), "mainnet", phase0.BLSPubKey{}, phase0.Root{}, createAttestationData(0, 1))
	require.Error(t, err)
}

// TestClient_CheckAttestation_DoubleVote tests cases where an attestation
// must be slashed because it is double voting.
// Borrowed from Prysm at https://github.com/prysmaticlabs/prysm/blob/a9a4bb9163da0e214797eadea847b046037ede6d/validator/db/kv/attester_protection_test.go#L45
func TestClient_CheckAttestation_DoubleVote(t *testing.T) {
	ctx := context.Background()
	client, _ := setupClient(t)

	tests := []struct {
		name                string
		existingAttestation *phase0.AttestationData
		existingSigningRoot [32]byte
		incomingAttestation *phase0.AttestationData
		incomingSigningRoot [32]byte
		want                bool
	}{
		{
			name:                "different signing root at same target equals a double vote",
			existingAttestation: createAttestationData(0, 1 /* Target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestationData(0, 1 /* Target */),
			incomingSigningRoot: [32]byte{2},
			want:                true,
		},
		{
			name:                "same signing root at same target is safe",
			existingAttestation: createAttestationData(0, 1 /* Target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestationData(0, 1 /* Target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
		{
			name:                "different signing root at different target is safe",
			existingAttestation: createAttestationData(0, 1 /* Target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestationData(0, 2 /* Target */),
			incomingSigningRoot: [32]byte{2},
			want:                false,
		},
		{
			name:                "no data stored at target should not be considered a double vote",
			existingAttestation: createAttestationData(0, 1 /* Target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestationData(0, 2 /* Target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check, err := client.CheckAttestation(
				ctx,
				"mainnet",
				phase0.BLSPubKey{},
				tt.existingSigningRoot,
				tt.existingAttestation,
			)
			require.NoError(t, err)
			require.False(t, check.Slashable, check.Reason)

			check2, err := client.CheckAttestation(
				ctx,
				"mainnet",
				phase0.BLSPubKey{},
				tt.incomingSigningRoot,
				tt.incomingAttestation,
			)
			require.NoError(t, err)
			if tt.want {
				require.True(t, check2.Slashable, check2.Reason)
				if !strings.Contains(check2.Reason, "double vote") &&
					!strings.Contains(check2.Reason, "could not sign attestation lower than or equal to lowest target epoch in db") {
					require.Fail(t, "unexpected reason: %s", check2.Reason)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_CheckProposal_Valid(t *testing.T) {
	client, _ := setupClient(t)
	check, err := client.CheckProposal(context.Background(), "mainnet", phase0.BLSPubKey{}, phase0.Root{}, 32)
	require.NoError(t, err)
	require.False(t, check.Slashable, "unexpected slashing: %s", check.Reason)
}

// setupClient creates a test client for testing.
func setupClient(t testing.TB) (*Client, *httptest.Server) {
	// Create a protector in a temporary directory.
	tempDir := t.TempDir()
	protector := protector.New(tempDir)

	// Create a test server.
	server := httptest.NewServer(NewServer(protector))

	t.Cleanup(func() {
		server.Close()
		require.NoError(t, protector.Close(), "failed to close protector")
	})

	return NewClient(http.DefaultClient, server.URL), server
}

func createAttestationData(sourceEpoch, targetEpoch phase0.Epoch) *phase0.AttestationData {
	return &phase0.AttestationData{
		Source: &phase0.Checkpoint{
			Epoch: sourceEpoch,
		},
		Target: &phase0.Checkpoint{
			Epoch: targetEpoch,
		},
	}
}
