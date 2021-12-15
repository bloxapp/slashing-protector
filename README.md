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
    spc "github.com/bloxapp/slashing-protector/client"
)

client := spc.New(&http.Client{})

// Check if an attestation is slashable:
resp, err := client.CheckAttestation(ctx, network, pubKey, signingRoot, &phase0.Attestation{...})
if err != nil {
    return err
}
if resp.Slashable {
    return errors.New("slashable attestation: %s", resp.Reason)
}
// <- Not slashable, can submit!

// Check if a proposal is slashable:
resp, err := client.CheckProposal(ctx, network, pubKey, signingRoot, &altair.BeaconBlock{...})
if err != nil {
    log.Fatal(err)
}
if resp.Slashable {
    return errors.New("slashable proposal: %s", resp.Reason)
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
3. 📦 Update the Go dependency to the commit hash:
    ```bash
    go get github.com/prysmaticlabs/prysm@<commit_hash>
    ```
4. 🧪 Run the tests to ensure no breaking changes.