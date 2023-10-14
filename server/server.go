// Package server provides utilities for working with Minecraft servers, including
// querying server status and sending Server List Ping (SLP) requests.
package server

import (
	"fmt"
	"time"

	"github.com/sch8ill/mclib/address"
	"github.com/sch8ill/mclib/slp"
)

// MCServer represents a Minecraft server with its address and client.
type MCServer struct {
	addr    *address.Address
	timeout time.Duration
	srv     bool
}

// MCServerOption represents a functional option for configuring an MCServer instance.
type MCServerOption func(*MCServer)

// WithTimeout sets a custom timeout for communication with the server.
func WithTimeout(timeout time.Duration) MCServerOption {
	return func(s *MCServer) {
		s.timeout = timeout
	}
}

// WithoutSRV disables SRV record resolution when creating an MCServer instance.
func WithoutSRV() MCServerOption {
	return func(s *MCServer) {
		s.srv = false
	}
}

// New creates a new MCServer instance with the provided raw address.
func New(rawAddr string, opts ...MCServerOption) (*MCServer, error) {
	addr, err := address.New(rawAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}

	s := &MCServer{
		addr:    addr,
		timeout: slp.DefaultTimeout,
		srv:     true,
	}
	for _, opt := range opts {
		opt(s)
	}

	if s.srv {
		_ = addr.ResolveSRV()
	}

	return s, nil
}

// StatusPing sends an SLP status ping to the Minecraft server and returns the response.
func (s *MCServer) StatusPing(opts ...slp.ClientOption) (*slp.Response, error) {
	client, err := s.createSLPClient(opts...)
	if err != nil {
		return nil, err
	}

	res, err := client.StatusPing()
	if err != nil {
		return nil, fmt.Errorf("status ping failed: %w", err)
	}

	return res, nil
}

// createSLPClient creates a new SLPClient instance with the provided options.
func (s *MCServer) createSLPClient(opts ...slp.ClientOption) (*slp.Client, error) {
	// the option is prepended so that it can be overwritten
	opts = append([]slp.ClientOption{slp.WithTimeout(s.timeout)}, opts...)

	client, err := slp.NewClient(s.addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to intialize SLP client: %w", err)
	}

	return client, nil
}
