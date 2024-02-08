// Package slp provides an SLP client for querying information about Minecraft servers
// using the Server List Ping (SLP) protocol.
package slp

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sch8ill/mclib/address"
	"github.com/sch8ill/mclib/slp/packet"
)

const (
	DefaultTimeout               = 5 * time.Second
	DefaultProtocolVersion int32 = 47

	handshakePacketID  int32 = 0
	statusPacketID     int32 = 0
	pingPacketID       int32 = 1
	pongPacketId       int32 = 1
	disconnectPacketID int32 = 27
	// the disconnect packet id has changed to 27 in 1.20.2
	legacyDisconnectPacketID int32 = 26

	statusState int32 = 1
)

// ConnState represents the connection state of the SLPClient.
type ConnState int64

const (
	Idle ConnState = iota
	Connected
	Handshaked
)

// Client represents an SLP client for interacting with Minecraft servers through the SLP protocol.
type Client struct {
	addr            *address.Address
	timeout         time.Duration
	protocolVersion int32
	state           ConnState
	conn            net.Conn
}

// ClientOption represents a functional option for configuring an SLPClient instance.
type ClientOption func(*Client)

// WithTimeout sets a custom timeout for communication with the server.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithProtocolVersion sets a custom Minecraft protocol version.
func WithProtocolVersion(version int32) ClientOption {
	return func(c *Client) {
		c.protocolVersion = version
	}
}

// NewClient creates a new Client for pinging a Minecraft server at the specified address.
func NewClient(addr *address.Address, opts ...ClientOption) (*Client, error) {
	client := &Client{
		addr:            addr,
		timeout:         DefaultTimeout,
		protocolVersion: DefaultProtocolVersion,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// StatusPing performs both a status query and a ping to the Minecraft server and returns the combined result.
func (c *Client) StatusPing() (*Response, error) {
	res, err := c.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get server status: %w", err)
	}

	latency, err := c.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to determine latency: %w", err)
	}
	res.Latency = latency

	return res, nil
}

// Status performs a status query to the Minecraft server and retrieves server information.
func (c *Client) Status() (*Response, error) {
	if err := c.connectAndHandshake(); err != nil {
		return nil, err
	}

	if err := c.sendStatusRequest(); err != nil {
		return nil, err
	}

	rawRes, err := c.recvResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to receive status response: %w", err)
	}

	res, err := NewResponse(rawRes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json response: %w", err)
	}

	return res, nil
}

// Ping performs a ping operation to the Minecraft server and returns the latency in milliseconds.
func (c *Client) Ping() (int, error) {
	if err := c.connectAndHandshake(); err != nil {
		return 0, err
	}

	timestamp := time.Now()

	if err := c.sendPing(timestamp.Unix()); err != nil {
		return 0, err
	}

	id, err := c.recvPong()
	if err != nil {
		return 0, fmt.Errorf("failed to receive pong: %w", err)
	}

	latency := int(time.Since(timestamp).Milliseconds())

	if id != timestamp.Unix() {
		return latency, fmt.Errorf("server responded with wrong pong id")
	}

	// the server closes the connection after the pong packet
	c.state = Idle
	return latency, nil
}

// sendHandshake sends a handshake packet to the Minecraft server during the connection setup.
func (c *Client) sendHandshake() error {
	// handshake packet:
	//		packet id          (VarInt) (0)
	//		protocol version   (VarInt) (-1 = not set)
	//		length of hostname (uint8)
	//		hostname           (string)
	//		port               (uint16)
	//		next state         (VarInt) (1 for status)
	//
	// https://wiki.vg/Server_List_Ping#Handshake

	handshake := packet.NewOutboundPacket(handshakePacketID)

	handshake.WriteVarInt(c.protocolVersion)
	if err := handshake.WriteString(c.addr.Host); err != nil {
		return fmt.Errorf("failed to write host: %w", err)
	}
	handshake.WriteShort(int16(c.addr.Port))
	handshake.WriteVarInt(statusState)
	if err := handshake.Write(c.conn); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	c.state = Handshaked

	return nil
}

// sendStatusRequest sends a status request packet to the Minecraft server.
func (c *Client) sendStatusRequest() error {
	// status request:
	//		packet id (VarInt) (0)
	//
	// https://wiki.vg/Protocol#Status_Request

	statusRequest := packet.NewOutboundPacket(statusPacketID)
	if err := statusRequest.Write(c.conn); err != nil {
		return fmt.Errorf("failed to send status request: %w", err)
	}

	return nil
}

// recvResponse receives the status response from the Minecraft server.
func (c *Client) recvResponse() (string, error) {
	// status response:
	//		packet id               (VarInt) (0)
	//		length of json response (uint8)
	//		json response           (string)
	//
	// https://wiki.vg/Server_List_Ping#Status_Response

	res, err := packet.NewInboundPacket(c.conn, c.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to read status response: %w", err)
	}

	id := res.ID()

	if id == disconnectPacketID || id == legacyDisconnectPacketID {
		msg, err := res.ReadString()
		if err != nil {
			return "", fmt.Errorf("failed to read disconnect reason: %w", err)
		}

		return "", fmt.Errorf("received disconnect packet from server: %s", msg)
	}

	if id != statusPacketID {
		return "", fmt.Errorf("response packet contains bad packet id: %d", res.ID())
	}

	resBody, err := res.ReadString()
	if err != nil {
		return "", fmt.Errorf("failed to read status response body: %w", err)
	}

	return resBody, nil
}

// sendPing sends a ping packet to the Minecraft server to measure latency.
func (c *Client) sendPing(timestamp int64) error {
	// ping packet:
	//		packet id (VarInt) (1)
	//		timestamp (Int64)
	//
	// https://wiki.vg/Server_List_Ping#Ping_Request

	ping := packet.NewOutboundPacket(pingPacketID)
	ping.WriteLong(timestamp)
	if err := ping.Write(c.conn); err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	return nil
}

// recvPong receives the pong packet from the Minecraft server.
func (c *Client) recvPong() (int64, error) {
	// pong packet:
	//		packet id (VarInt) (1)
	//		payload   (Int64)
	//
	// https://wiki.vg/Server_List_Ping#Pong_Response

	pong, err := packet.NewInboundPacket(c.conn, c.timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to read pong: %w", err)
	}

	if pong.ID() != pongPacketId {
		return 0, fmt.Errorf("response packet contains bad packet id: %d", pong.ID())
	}

	id, err := pong.ReadLong()
	if err != nil {
		return 0, fmt.Errorf("failed to read pong id: %w", err)
	}

	return id, nil
}

// connectAndHandshake handles the connection setup and handshake with the Minecraft server.
func (c *Client) connectAndHandshake() error {
	if c.state < Connected {
		if err := c.connect(); err != nil {
			return err
		}
	}

	if c.state < Handshaked {
		if err := c.sendHandshake(); err != nil {
			return err
		}
	}

	return nil
}

// connect establishes a connection to the Minecraft server.
func (c *Client) connect() error {
	if c.state > Idle {
		return errors.New("client is already connected")
	}

	conn, err := net.DialTimeout("tcp", c.addr.Addr(), c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	c.state = Connected

	return nil
}
