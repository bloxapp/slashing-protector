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

// fileName returns the database filename of the connection.
func (id connID) fileName() string {
	return fmt.Sprintf("kvstore-%s-%x", id.network, id.pubKey)
}

// Pool implements a kv.Store pool with a single connection per public key in a network.
type Pool struct {
	dir    string
	conn   map[connID]*Conn
	poolMu sync.Mutex
}

func New(dir string) *Pool {
	return &Pool{
		dir:  dir,
		conn: make(map[connID]*Conn),
	}
}

// Acquire returns a connection from the pool, creating one if necessary.
// The caller must call Release() when the connection is no longer needed.
func (p *Pool) Acquire(
	ctx context.Context,
	network string,
	pubKey phase0.BLSPubKey,
) (*Conn, error) {
	conn := p.getOrCreate(connID{network, pubKey})
	if err := conn.acquire(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}

// getOrCreate returns a connection from the pool, creating one if necessary.
func (p *Pool) getOrCreate(id connID) *Conn {
	p.poolMu.Lock()
	defer p.poolMu.Unlock()

	if conn, ok := p.conn[id]; ok {
		// Return existing connection.
		return conn
	}

	// Create the connection.
	fileName := filepath.Join(p.dir, id.fileName())
	conn := newConn(fileName)
	p.conn[id] = conn
	return conn
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.poolMu.Lock()
	defer p.poolMu.Unlock()
	for _, c := range p.conn {
		if err := c.Store.Close(); err != nil {
			return err
		}
	}
	p.conn = make(map[connID]*Conn)
	return nil
}
