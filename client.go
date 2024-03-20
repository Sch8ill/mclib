package mclib

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sch8ill/mclib/address"
	"github.com/sch8ill/mclib/packet"
	"github.com/sch8ill/mclib/slp"
)

const (
	DefaultTimeout        = 5 * time.Second
	DefaultProtocol int32 = 47

	StatusState int32 = 1
	LoginState  int32 = 2
)

// ConnState represents the connection state of the Client.
type ConnState int64

const (
	Idle ConnState = iota
	Connected
	HandshakeComplete
)

// Client represents a client for interacting with Minecraft servers through the Minecraft protocol.
type Client struct {
	addr     *address.Address
	timeout  time.Duration
	srv      bool
	protocol int32
	state    ConnState
	conn     net.Conn
}

// ClientOption represents a functional option for configuring a Client instance.
type ClientOption func(*Client)

// WithTimeout sets a custom timeout for communication with the server.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithProtocolVersion sets a custom Minecraft protocol version.
func WithProtocolVersion(protocol int32) ClientOption {
	return func(c *Client) {
		c.protocol = protocol
	}
}

// WithoutSRV disables SRV record lookups for the client.
func WithoutSRV() ClientOption {
	return func(c *Client) {
		c.srv = false
	}
}

// WithConnection set a custom already connected connection.
func WithConnection(conn net.Conn) ClientOption {
	return func(c *Client) {
		c.conn = conn
		c.state = Connected
	}
}

// WithAddress sets a custom address.
func WithAddress(addr *address.Address) ClientOption {
	return func(c *Client) {
		c.addr = addr
	}
}

// NewClient creates a new Client for pinging a Minecraft server at the specified address.
func NewClient(addr string, opts ...ClientOption) (*Client, error) {
	a, err := address.New(addr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		addr:     a,
		timeout:  DefaultTimeout,
		protocol: DefaultProtocol,
		srv:      true,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// StatusPing performs both a status query and a ping to the Minecraft server and returns the combined result.
func (c *Client) StatusPing() (*slp.Response, error) {
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
func (c *Client) Status() (*slp.Response, error) {
	if err := c.connectAndHandshake(StatusState); err != nil {
		return nil, err
	}

	if err := c.sendStatusRequest(); err != nil {
		return nil, err
	}

	rawRes, err := c.recvStatusResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to receive status response: %w", err)
	}

	res, err := slp.NewResponse(rawRes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json response: %w", err)
	}

	return res, nil
}

// Ping performs a ping operation to the Minecraft server and returns the latency in milliseconds.
func (c *Client) Ping() (int, error) {
	if err := c.connectAndHandshake(StatusState); err != nil {
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

// LoginError tries to trigger an exception in the servers packet parser.
// The error response can be used to fingerprint the server software.
func (c *Client) LoginError() (string, int32, error) {
	if err := c.connectAndHandshake(LoginState); err != nil {
		return "", 0, err
	}

	if err := c.sendLoginStartCrash("mclib", make([]byte, 16)); err != nil {
		return "", 0, err
	}

	res, err := packet.NewInboundPacket(c.conn, c.timeout)
	if err != nil {
		return "", 0, err
	}

	reason, err := res.ReadString()
	if err != nil {
		return "", 0, err
	}

	return reason, res.ID(), nil
}

// sendHandshake sends a handshake packet to the Minecraft server during the connection setup.
func (c *Client) sendHandshake(state int32) error {
	// handshake packet:
	//		packet id        (VarInt) (0)
	//		protocol version (VarInt) (-1 = not set)
	//		hostname         (string)
	//		port             (uint16)
	//		next state       (VarInt) (1: status, 2: login)
	//
	// https://wiki.vg/Server_List_Ping#Handshake

	handshake := packet.NewOutboundPacket(packet.HandshakeID)
	handshake.WriteVarInt(c.protocol)
	if err := handshake.WriteString(c.addr.Host()); err != nil {
		return fmt.Errorf("failed to write host: %w", err)
	}
	handshake.WriteShort(int16(c.addr.Port()))
	handshake.WriteVarInt(state)
	if err := handshake.Write(c.conn); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	c.state = HandshakeComplete

	return nil
}

// sendStatusRequest sends a status request packet to the Minecraft server.
func (c *Client) sendStatusRequest() error {
	// status request:
	//		packet id (VarInt) (0)
	//
	// https://wiki.vg/Protocol#Status_Request

	statusRequest := packet.NewOutboundPacket(packet.StatusID)
	if err := statusRequest.Write(c.conn); err != nil {
		return fmt.Errorf("failed to send status request: %w", err)
	}

	return nil
}

// recvResponse receives the status response from the Minecraft server.
func (c *Client) recvStatusResponse() (string, error) {
	// status response:
	//		packet id     (VarInt) (0)
	//		json response (string)
	//
	// https://wiki.vg/Server_List_Ping#Status_Response

	res, err := packet.NewInboundPacket(c.conn, c.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to read status response: %w", err)
	}

	id := res.ID()
	if id == packet.DisconnectID || id == packet.LegacyDisconnectID {
		msg, err := res.ReadString()
		if err != nil {
			return "", fmt.Errorf("failed to read disconnect reason: %w", err)
		}

		return "", fmt.Errorf("disconnect packet from server: %s", msg)
	}

	if id != packet.StatusID {
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

	ping := packet.NewOutboundPacket(packet.PingID)
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

	if pong.ID() != packet.PongID {
		return 0, fmt.Errorf("response packet contains bad packet id: %d", pong.ID())
	}

	id, err := pong.ReadLong()
	if err != nil {
		return 0, fmt.Errorf("failed to read pong id: %w", err)
	}

	return id, nil
}

// sendLoginStartCrash sends a bad login start packet to the server to trigger an error.
func (c *Client) sendLoginStartCrash(name string, uuid []byte) error {
	// login start crash packet:
	//		packet id (VarInt) (0)
	//		name      (string)
	//		uuid      (uuid)
	//
	// unexpected:
	//		padding (byte)
	//
	// https://wiki.vg/Protocol#Login_Start

	if len(name) > 16 {
		return fmt.Errorf("player name cannot be longer than 16 characters: length: %d", len(name))
	}

	if len(uuid) != 16 {
		return fmt.Errorf("player uuid has to be 16 bytes long: length: %d", len(uuid))
	}

	login := packet.NewOutboundPacket(packet.LoginStartID)
	if err := login.WriteString(name); err != nil {
		return err
	}
	login.WriteBytes(uuid)
	login.WriteByte(0)

	if err := login.Write(c.conn); err != nil {
		return err
	}

	return nil
}

// connectAndHandshake handles the connection setup and handshake with the Minecraft server.
func (c *Client) connectAndHandshake(state int32) error {
	if c.state < Connected {
		if err := c.connect(); err != nil {
			return err
		}
	}

	if c.state < HandshakeComplete {
		if err := c.sendHandshake(state); err != nil {
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

	if c.srv {
		_ = c.addr.ResolveSRV()
	}

	conn, err := net.DialTimeout("tcp", c.addr.String(), c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	c.state = Connected

	return nil
}
