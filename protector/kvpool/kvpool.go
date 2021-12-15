package kvpool

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"golang.org/x/sync/semaphore"
)

// Conn is a connection acquired from the pool.
type Conn struct {
	*kv.Store
	fileName  string
	semaphore *semaphore.Weighted
}

func newConn(fileName string) *Conn {
	return &Conn{
		fileName:  fileName,
		semaphore: semaphore.NewWeighted(1),
	}
}

func (c *Conn) acquire(ctx context.Context) error {
	if err := c.semaphore.Acquire(ctx, 1); err != nil {
		return err
	}
	store, err := kv.NewKVStore(
		ctx,
		c.fileName,
		&kv.Config{},
	)
	if err != nil {
		return err
	}
	c.Store = store
	return nil
}

// Release returns a connection to the pool.
func (c *Conn) Release() error {
	if err := c.Store.Close(); err != nil {
		return err
	}
	c.semaphore.Release(1)
	return nil
}

// connID is a unique identifier for a connection.
type connID struct {
	network string
	pubKey  phase0.BLSPubKey
}

func (id connID) fileName() string {
	return fmt.Sprintf("%s-%x", id.network, id.pubKey)
}

// Pool implements a kv.Store pool with a single connection per public key in a network.
type Pool struct {
	dir    string
	conn   map[connID]*Conn
	poolMu sync.RWMutex
}

func New(dir string) *Pool {
	return &Pool{
		dir:  dir,
		conn: make(map[connID]*Conn),
	}
}

// Acquire returns a connection from the pool, creating one if necessary.
// The caller must call Release() when the connection is no longer needed.
func (p *Pool) Acquire(ctx context.Context, network string, pubKey phase0.BLSPubKey) (*Conn, error) {
	// Search for an existing connection for this network and public key.
	id := connID{network: network, pubKey: pubKey}
	p.poolMu.RLock()
	c, ok := p.conn[id]
	p.poolMu.RUnlock()
	if ok {
		// Connection exists, wait for it to become available
		// and return it.
		if err := c.acquire(ctx); err != nil {
			return nil, err
		}
		return c, nil
	}

	// Create a new connection, add it to the pool
	// and return it.
	c = newConn(filepath.Join(p.dir, id.fileName()))
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}

	p.poolMu.Lock()
	p.conn[id] = c
	p.poolMu.Unlock()

	return c, nil
}
