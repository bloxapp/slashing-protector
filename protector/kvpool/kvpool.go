package kvpool

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

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
		if err := c.Release(); err != nil {
			if err == ErrConnNotAcquired {
				continue
			}
			return errors.Wrap(err, "Conn.Release")
		}
	}
	p.conn = make(map[connID]*Conn)
	return nil
}

// AcquiredConns returns the number of connections currently acquired.
func (p *Pool) AcquiredConns() int {
	p.poolMu.Lock()
	defer p.poolMu.Unlock()
	var n int
	for _, c := range p.conn {
		if c.Store != nil {
			n++
		}
	}
	return n
}
