package kvpool

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"golang.org/x/sync/semaphore"
)

// Conn is a connection acquired from the pool.
type Conn struct {
	*kv.Store
	fileName       string
	semaphore      *semaphore.Weighted
	cancelStoreCtx func()
}

func newConn(fileName string) *Conn {
	return &Conn{
		fileName:  fileName,
		semaphore: semaphore.NewWeighted(1),
	}
}

func (c *Conn) acquire(ctx context.Context) (err error) {
	if err := c.semaphore.Acquire(ctx, 1); err != nil {
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	defer func() {
		if err != nil {
			c.semaphore.Release(1)
		}
	}()

	// kv.NewKVStore starts a background goroutine which only stops when the
	// context is cancelled. However, cancelling the context before
	// Store is closed causes some methods (such as SaveAttestationForPubKey)
	// to hang forever.
	// Therefore, we create a context and cancel it only after Store is closed.
	ctxStore, cancelStore := context.WithCancel(context.Background())
	c.cancelStoreCtx = cancelStore
	store, err := kv.NewKVStore(
		ctxStore,
		c.fileName,
		&kv.Config{},
	)
	if err != nil {
		// dirty hack alert: Ignore this prometheus error as we are opening two DB with same metric name
		// if you want to avoid this then we should pass the metric name when opening the DB which touches
		// too many places.
		// Borrowed from Prysm at https://github.com/prysmaticlabs/prysm/blob/29513c804caad88cf4e93eefdde0d71ea9eb6e75/tools/exploredb/main.go#L390-L395
		if err.Error() != "duplicate metrics collector registration attempted" {
			return fmt.Errorf("kv.NewKVStore(%s): %w", c.fileName, err)
		}
	}
	c.Store = store
	return nil
}

// Release returns the connection to the connection pool.
func (c *Conn) Release() error {
	defer c.semaphore.Release(1)
	if c.cancelStoreCtx != nil {
		defer c.cancelStoreCtx()
	}
	if c.Store == nil {
		return errors.New("connection not acquired")
	}
	if err := c.Store.Close(); err != nil {
		return errors.Wrap(err, "kv.Store.Close")
	}
	c.Store = nil
	return nil
}
