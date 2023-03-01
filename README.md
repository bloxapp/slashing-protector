# slashing-protector

`slashing-protector` provides an HTTP API to protect Ethereum 2.0 validators from slashing.

## Setting up a server

Clone this repository:
```
git clone github.com/bloxapp/slashing-protector
```

Run with Docker:
```
cd slashing-protector
docker-compose up
```

API is ready for use at http://localhost:9369 🤙

## Client usage

Use the `client` package to interact with the `slashing-protector` API:

```go
import (
    "log"
    "net/http"
    sp "github.com/bloxapp/slashing-protector/http"
)

client := sp.NewClient(&http.Client{})

// Check if an attestation is slashable:
check, err := client.CheckAttestation(ctx, network, pubKey, signingRoot, &phase0.Attestation{...})
if err != nil {
    return err
}
if check.Slashable {
    return errors.New("slashable attestation: %s", check.Reason)
}
// <- Not slashable, can submit!

// Check if a proposal is slashable:
check, err := client.CheckProposal(ctx, network, pubKey, signingRoot, &altair.BeaconBlock{...})
if err != nil {
    return err
}
if check.Slashable {
    return errors.New("slashable proposal: %s", check.Reason)
}
// <- Not slashable, can submit!
```

## Developer guide

`slashing-protector` leverages Prysm's [bbolt](https://github.com/etcd-io/bbolt)-based slashing protection. (See https://github.com/prysmaticlabs/prysm/tree/v2.0.4/validator/db/kv)

This means that when Prysm releases a version, the dependency might need to be updated. Specifically, if the release fixes a bug or implements a specification change regarding slashing protection, then the dependency **must** be updated.

A good practice would be to update the dependency with every Prysm stable release.

### Updating the Prysm dependency

1. #️⃣ Copy the commit hash of the Prysm release or hotfix
2. 📁 Navigate to `slashing-protector`'s directory
3. 📦 Edit `go.mod` and paste the first 12 characters of the commit hash in the Prysm dependency:
    ```
    replace github.com/prysmaticlabs/prysm v1.4.4 => github.com/prysmaticlabs/prysm v1.4.2-0.20211005004110-<short_hash>
    ```
4. 🧹 Run `go mod tidy` and it'll tell you that the timestamp (right before the hash) is wrong and what it should be instead. Copy the correct timestamp and update it in `go.mod`. Run `go mod tidy` again to ensure everything's okay.
5. 📕 Read the TODO above [CheckAttestation](https://github.com/bloxapp/slashing-protector/blob/40a87a234d5e17280ae1a023b90b34f74065871e/protector/protector.go#L109-L205), and if needed, update accordingly!
6. 🧪 Run the tests to ensure no breaking changes.

### Testing

```bash
go test ./...
```

👆 Nothin' fancy.