// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oisee/open-rfc-go/internal/client"
	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/pool"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/saprouter"
	"github.com/oisee/open-rfc-go/internal/socks5"
)

// SOCKS5Options configures a SOCKS5 proxy in front of the connection. Set
// JWTToken for SAP BTP Connectivity (method 0x80), or Username/Password for
// RFC 1929; leave both unset for no-auth.
type SOCKS5Options struct {
	ProxyAddress string
	Username     string
	Password     string
	JWTToken     string
	LocationID   string
}

// PoolConfig bounds the session pool. Zero values select sensible defaults.
type PoolConfig struct {
	MaxSize        int
	MaxIdleTime    time.Duration
	AcquireTimeout time.Duration
}

// Destination describes how to reach and log on to a system.
type Destination struct {
	Host             string // gateway host
	Port             int    // gateway port (e.g. 3300 for instance 00)
	Client           string // ABAP client, e.g. "001"
	User             string
	Password         string
	Language         string // default "E"
	Service          string // dispatcher service name, sapdpNN
	ProgramName      string // default "open-rfc"
	Router           string // SAProuter route prefix ending in /H/ (optional)
	SOCKS5           *SOCKS5Options
	Pool             PoolConfig
	OperationTimeout time.Duration
	// Callbacks handles server-initiated RFC callbacks (a called FM doing
	// CALL FUNCTION … DESTINATION 'BACK') by function-module name. Leave nil if
	// the destination is never expected to call back.
	Callbacks map[string]CallbackFunc
}

// Client is a pooled, authenticated connection to one Destination. It is safe
// for concurrent use.
type Client struct {
	pool *pool.Pool[*lifecycle.Managed]
	dest Destination

	mu          sync.Mutex
	fnCache     map[string]metadata.RfcFunctionInterface
	structCache map[string]rfctypes.RfcStructureDefinition
	// graphCache holds normalized RFC_METADATA_GET type graphs (and the
	// failure to obtain one) per function module — see recursive.go.
	graphCache map[string]graphEntry
}

// Open dials the destination, establishes a pool of authenticated sessions, and
// returns a ready client. It fails fast by opening one session eagerly.
func Open(ctx context.Context, d Destination) (*Client, error) {
	if d.Host == "" || d.User == "" {
		return nil, fmt.Errorf("%w: Host and User are required", ErrProtocol)
	}
	openSession := func(ctx context.Context) (*lifecycle.Managed, error) {
		sopts := client.SessionOptions{
			Host:                     d.Host,
			Port:                     d.Port,
			ApplicationServerService: d.Service,
			ProgramName:              d.ProgramName,
			OperationTimeout:         d.OperationTimeout,
		}
		if d.Router != "" {
			route, err := saprouter.CompleteRoute(d.Router, d.Host, d.Port)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrProtocol, err)
			}
			sopts.Router = route
		}
		if d.SOCKS5 != nil {
			sopts.Proxy = socks5Dialer(d.SOCKS5)
		}
		sess, err := client.Open(ctx, sopts)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTransport, err)
		}
		if err := sess.LogonAndPing(ctx, client.LogonOptions{
			Client:   d.Client,
			User:     d.User,
			Password: d.Password,
			Language: d.Language,
		}); err != nil {
			_ = sess.Close()
			return nil, fmt.Errorf("%w: %v", ErrLogonRejected, err)
		}
		return lifecycle.Wrap(sess), nil
	}

	p, err := lifecycle.NewPool(lifecycle.PoolOptions{
		Open: openSession,
		Pool: pool.Config{
			MaxSize:        d.Pool.MaxSize,
			MaxIdleTime:    d.Pool.MaxIdleTime,
			AcquireTimeout: d.Pool.AcquireTimeout,
		},
	})
	if err != nil {
		return nil, err
	}
	c := &Client{
		pool:        p,
		dest:        d,
		fnCache:     map[string]metadata.RfcFunctionInterface{},
		structCache: map[string]rfctypes.RfcStructureDefinition{},
		graphCache:  map[string]graphEntry{},
	}
	// Fail fast: prove the destination works before returning.
	lease, err := p.Acquire(ctx)
	if err != nil {
		_ = p.Close(context.Background())
		return nil, err
	}
	lease.Release()
	return c, nil
}

func socks5Dialer(o *SOCKS5Options) *socks5.Dialer {
	var auth socks5.Auth
	switch {
	case o.JWTToken != "":
		auth = socks5.SAPJWT{Token: o.JWTToken, LocationID: o.LocationID}
	case o.Username != "":
		auth = socks5.UserPass{Username: o.Username, Password: o.Password}
	}
	return &socks5.Dialer{ProxyAddress: o.ProxyAddress, Auth: auth}
}

// Close shuts the client and its pool down.
func (c *Client) Close(ctx context.Context) error { return c.pool.Close(ctx) }

// FunctionInterface returns a function module's interface, cached per client.
func (c *Client) FunctionInterface(ctx context.Context, name string) (metadata.RfcFunctionInterface, error) {
	if v, ok := c.cachedFn(name); ok {
		return v, nil
	}
	lease, err := c.pool.Acquire(ctx)
	if err != nil {
		return metadata.RfcFunctionInterface{}, translate(err)
	}
	defer lease.Release()
	return c.functionInterfaceOn(ctx, lease.Value(), name)
}

// StructureDefinition returns a DDIC structure's layout, cached per client.
func (c *Client) StructureDefinition(ctx context.Context, name string) (rfctypes.RfcStructureDefinition, error) {
	if v, ok := c.cachedStruct(name); ok {
		return v, nil
	}
	lease, err := c.pool.Acquire(ctx)
	if err != nil {
		return rfctypes.RfcStructureDefinition{}, translate(err)
	}
	defer lease.Release()
	return c.structureDefinitionOn(ctx, lease.Value(), name)
}

func (c *Client) cachedFn(name string) (metadata.RfcFunctionInterface, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.fnCache[name]
	return v, ok
}

func (c *Client) cachedStruct(name string) (rfctypes.RfcStructureDefinition, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.structCache[name]
	return v, ok
}

func (c *Client) functionInterfaceOn(ctx context.Context, sess *lifecycle.Managed, name string) (metadata.RfcFunctionInterface, error) {
	if v, ok := c.cachedFn(name); ok {
		return v, nil
	}
	req, err := metadata.BuildRfcGetFunctionInterfaceRequest(name)
	if err != nil {
		return metadata.RfcFunctionInterface{}, err
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		return metadata.RfcFunctionInterface{}, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	if exc := exceptionFromEnvelope(res.Envelope); exc != nil {
		return metadata.RfcFunctionInterface{}, exc
	}
	iface, err := metadata.DecodeRfcFunctionInterfaceResult(name, res.Fields)
	if err != nil {
		return metadata.RfcFunctionInterface{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	c.mu.Lock()
	c.fnCache[name] = iface
	c.mu.Unlock()
	return iface, nil
}

func (c *Client) structureDefinitionOn(ctx context.Context, sess *lifecycle.Managed, name string) (rfctypes.RfcStructureDefinition, error) {
	if v, ok := c.cachedStruct(name); ok {
		return v, nil
	}
	req, err := metadata.BuildRfcGetStructureDefinitionRequest(name)
	if err != nil {
		return rfctypes.RfcStructureDefinition{}, err
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		return rfctypes.RfcStructureDefinition{}, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	if exc := exceptionFromEnvelope(res.Envelope); exc != nil {
		return rfctypes.RfcStructureDefinition{}, exc
	}
	def, err := metadata.DecodeRfcStructureDefinitionResult(name, res.Fields)
	if err != nil {
		// A table type resolves via RFC_FIELDS to its row (line) structure, whose
		// fields belong to that row, not the queried table-type name. Dereference
		// to the row structure and resolve that instead.
		if row, rerr := metadata.RowStructureName(name, res.Fields); rerr == nil && row != "" && row != name {
			rdef, rerr2 := c.structureDefinitionOn(ctx, sess, row)
			if rerr2 == nil {
				c.mu.Lock()
				c.structCache[name] = rdef
				c.mu.Unlock()
				return rdef, nil
			}
		}
		return rfctypes.RfcStructureDefinition{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	c.mu.Lock()
	c.structCache[name] = def
	c.mu.Unlock()
	return def, nil
}
