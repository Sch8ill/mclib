// Package packet provides utilities for sending and receiving Minecraft network packets.
package packet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// InboundPacket represents a packet received from a connection.
type InboundPacket struct {
	id     int32
	body   []byte
	reader *bufio.Reader
}

// NewInboundPacket creates a new InboundPacket from a network connection.
func NewInboundPacket(conn net.Conn) (*InboundPacket, error) {
	p := &InboundPacket{}
	connReader := bufio.NewReader(conn)

	uLength, err := binary.ReadUvarint(connReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet length: %w", err)
	}
	length := int(uLength)

	if length > MaxPacketLength {
		return nil, fmt.Errorf("received packet is too long: %d", length)
	}

	p.body, err = readBytes(connReader, length)
	if err != nil {
		return nil, fmt.Errorf("failed to receive packet body: %w", err)
	}

	p.reader = bufio.NewReader(bytes.NewReader(p.body))

	packetID, err := binary.ReadUvarint(p.reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet id: %w", err)
	}
	p.id = int32(packetID)

	return p, nil
}

// ReadInt reads a 32-bit integer from the packet.
func (p *InboundPacket) ReadInt() (int32, error) {
	buf := make([]byte, 4)

	_, err := io.ReadFull(p.reader, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read int: %w", err)
	}
	n := int32(binary.BigEndian.Uint32(buf))

	return n, nil
}

// ReadShort reads a 16-bit integer from the packet.
func (p *InboundPacket) ReadShort() (int16, error) {
	buf := make([]byte, 8)

	_, err := io.ReadFull(p.reader, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read short: %w", err)
	}
	n := int16(binary.BigEndian.Uint16(buf))

	return n, nil
}

// ReadLong reads a 64-bit integer from the packet.
func (p *InboundPacket) ReadLong() (int64, error) {
	buf := make([]byte, 8)

	_, err := io.ReadFull(p.reader, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read long: %w", err)
	}
	n := int64(binary.BigEndian.Uint64(buf))

	return n, nil
}

// ReadVarInt reads a variable-length 32-bit integer from the packet.
func (p *InboundPacket) ReadVarInt() (int32, error) {
	n, err := binary.ReadUvarint(p.reader)
	if err != nil {
		return 0, err
	}

	return int32(n), nil
}

// ReadVarLong reads a variable-length 64-bit integer from the packet.
func (p *InboundPacket) ReadVarLong() (int64, error) {
	n, err := p.ReadVarInt()
	if err != nil {
		return 0, err
	}

	return int64(n), err
}

// ReadBool reads a boolean value from the packet.
func (p *InboundPacket) ReadBool() (bool, error) {
	value, err := p.ReadByte()
	if err != nil {
		return false, fmt.Errorf("failed to read bool: %w", err)
	}

	return value != 0, nil
}

// ReadString reads a string from the packet.
func (p *InboundPacket) ReadString() (string, error) {
	uLength, err := p.ReadVarInt()
	if err != nil {
		return "", fmt.Errorf("failed to read string length: %w", err)
	}
	length := int(uLength)

	if length > MaxStringLength {
		return "", fmt.Errorf("received string exceeds the max string length: %d", length)
	}

	raw, err := p.ReadBytes(length)
	if err != nil {
		return "", fmt.Errorf("failed to read string: %w", err)
	}

	return string(raw), nil
}

// ReadByte reads a single byte from the packet.
func (p *InboundPacket) ReadByte() (byte, error) {
	buf, err := p.ReadBytes(1)
	if err != nil {
		return 0, fmt.Errorf("failed to read byte: %w", err)
	}

	return buf[0], nil
}

// ReadBytes reads a specified number of bytes from the packet.
func (p *InboundPacket) ReadBytes(length int) ([]byte, error) {
	b, err := readBytes(p.reader, length)
	if err != nil {
		return nil, fmt.Errorf("failed to read bytes: %w", err)
	}

	return b, nil
}

// ID returns the id of the packet.
func (p *InboundPacket) ID() int32 {
	return p.id
}

// readBytes reads a specified number of bytes from a buffered reader.
func readBytes(reader *bufio.Reader, length int) ([]byte, error) {
	data := make([]byte, length)
	var received int
	for received < length {
		segmentLength, err := reader.Read(data[received:])
		if err != nil {
			return nil, fmt.Errorf("failed to read packet segment: %w", err)
		}

		received += segmentLength
	}

	return data, nil
}
