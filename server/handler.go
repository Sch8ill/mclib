package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sch8ill/mclib"
	"github.com/sch8ill/mclib/packet"
	"github.com/sch8ill/mclib/slp"
)

type Handler struct {
	Address   net.Addr
	conn      net.Conn
	timeout   time.Duration
	handshake *handshake
}

type handshake struct {
	protocol  int32
	hostname  string
	port      int16
	nextState int32
}

func NewHandler(conn net.Conn, timeout time.Duration) *Handler {
	return &Handler{
		conn:    conn,
		Address: conn.RemoteAddr(),
		timeout: timeout,
	}
}

func (h *Handler) Handle() error {
	if err := h.handleHandshake(); err != nil {
		return err
	}

	log.Printf("%s: handshake: %+v", h.Address.String(), *h.handshake)

	switch h.handshake.nextState {
	case mclib.StatusState:
		if err := h.handleStatus(); err != nil {
			return err
		}

	case mclib.LoginState:
		player, err := h.handleLogin()
		if err != nil {
			return err
		}
		log.Printf("%s: login: %+v", h.Address.String(), *player)

	default:
		return nil
	}

	return nil
}

func (h *Handler) handleHandshake() error {
	p, err := packet.NewInboundPacket(h.conn, h.timeout)
	if err != nil {
		return fmt.Errorf("failed to receive handshake packet: %w", err)
	}

	if p.ID() != packet.HandshakeID {
		return fmt.Errorf("handshake packet id mismatch, expected %d, got %d", packet.HandshakeID, p.ID())
	}

	h.handshake = &handshake{}
	h.handshake.protocol, err = p.ReadVarInt()
	if err != nil {
		return fmt.Errorf("failed to read client protocol version: %w", err)
	}

	h.handshake.hostname, err = p.ReadString()
	if err != nil {
		return fmt.Errorf("failed to read hostname: %w", err)
	}

	h.handshake.port, err = p.ReadShort()
	if err != nil {
		return fmt.Errorf("failed to read host port: %w", err)
	}

	h.handshake.nextState, err = p.ReadVarInt()
	if err != nil {
		return fmt.Errorf("failed to read next state: %w", err)
	}

	return nil
}

func (h *Handler) handleStatus() error {
	p, err := packet.NewInboundPacket(h.conn, h.timeout)
	if err != nil {
		return fmt.Errorf("failed to receive status request packet: %w", err)
	}

	switch p.ID() {
	case packet.StatusID:
		if err := h.sendStatusResponse(); err != nil {
			return err
		}
		// listen for optional ping after status request
		h.handleStatus()

	case packet.PingID:
		if err := h.handlePing(p); err != nil {
			return err
		}

	default:
		return fmt.Errorf("status state packet id mismatch, expected %d or %d, got %d", packet.StatusID, packet.PingID, p.ID())
	}

	return nil
}

func (h *Handler) sendStatusResponse() error {
	p := packet.NewOutboundPacket(packet.StatusID)
	res := slp.Response{
		Description: slp.Description{Description: slp.ChatComponent{Text: "github.com/sch8ill/mclib"}},
		Players: slp.Players{
			Online: 3,
			Max:    20,
		},
		Version: slp.Version{
			Name:     "github.com/sch8ill/mclib",
			Protocol: 762,
		},
	}

	body, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("failed to marshal json response: %w", err)
	}
	p.WriteString(string(body))

	if err := p.Write(h.conn); err != nil {
		return fmt.Errorf("failed to send status response: %w", err)
	}

	log.Printf("%s: status request", h.Address.String())
	return nil
}

func (h *Handler) handlePing(ping *packet.InboundPacket) error {
	token, err := ping.ReadLong()
	if err != nil {
		return fmt.Errorf("failed to read ping token: %w", err)
	}

	pong := packet.NewOutboundPacket(packet.PongID)
	pong.WriteLong(token)

	if err := pong.Write(h.conn); err != nil {
		return fmt.Errorf("failed to send pong: %w", err)
	}

	log.Printf("%s: ping: token: %d", h.Address.String(), token)
	return nil
}

func (h *Handler) handleLogin() (*slp.Player, error) {
	start, err := packet.NewInboundPacket(h.conn, h.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to receive login start packet: %w", err)
	}

	if start.ID() != packet.LoginStartID {
		return nil, fmt.Errorf("login start packet id mismatch, expected %d, got %d", packet.LoginStartID, start.ID())
	}

	player := &slp.Player{}
	player.Name, err = start.ReadString()
	if err != nil {
		return nil, fmt.Errorf("failed to read player name: %w", err)
	}

	player.ID, err = start.ReadString()
	if err != nil {
		return nil, fmt.Errorf("failed to read play id %w", err)
	}

	if err := h.sendDisconnect(packet.LoginDisconnectID, "login not supported"); err != nil {
		return nil, err
	}

	return player, nil
}

func (h *Handler) sendDisconnect(id int32, msg string) error {
	p := packet.NewOutboundPacket(id)
	p.WriteString(msg)

	if err := p.Write(h.conn); err != nil {
		return fmt.Errorf("failed to send disconnect packet: %w", err)
	}

	return nil
}
