// Package fingerprint provides functionality to determine a Minecraft Servers software
// by sending maliciously crafted packets to the server and analyzing the responses.
package fingerprint

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sch8ill/mclib"
	"github.com/sch8ill/mclib/packet"
)

const (
	Vanilla     string = "vanilla"
	CraftBukkit        = "craftbukkit"
	Paper              = "paper"
	Fabric             = "fabric"
	Forge              = "forge"
	Velocity           = "velocity"
	Empty              = "empty"
	Encryption         = "encryption"
	Success            = "success"
	Compression        = "compression"
	Plugin             = "plugin"
	Unknown            = "unknown"
)

var ConnectionThrottled = errors.New("connection throttled by server")

func Fingerprint(addr string, opts ...mclib.ClientOption) (string, error) {
	statusClient, err := mclib.NewClient(addr, opts...)
	if err != nil {
		return Unknown, err
	}

	status, err := statusClient.Status()
	if err != nil {
		return Unknown, err
	}

	return FingerprintWithProtocol(addr, status.Version.Protocol, opts...)
}

func FingerprintWithProtocol(addr string, protocol int, opts ...mclib.ClientOption) (string, error) {
	opts = append(opts, mclib.WithProtocolVersion(int32(protocol)))
	client, err := mclib.NewClient(addr, opts...)
	if err != nil {
		return Unknown, fmt.Errorf("client creation failed: %w", err)
	}

	res, id, err := client.LoginError()
	if errors.Is(err, io.EOF) {
		return Empty, nil
	}
	if err != nil {
		return Unknown, err
	}

	if id != packet.LoginDisconnectID {
		return determineServerState(id)
	}

	// response is not json
	if !strings.HasPrefix(res, "{") {
		return parseErrorResponse(res)
	}

	msg, err := NewDisconnectMsg(res)
	if err != nil {
		return Unknown, err
	}

	mismatch, version := msg.VersionMismatch()
	if mismatch {
		return Unknown, fmt.Errorf("version mismatch: %s", version)
	}

	return msg.Fingerprint()
}

func determineServerState(id int32) (string, error) {
	switch id {
	case packet.LoginEncryptionID:
		return Encryption, nil

	case packet.LoginSuccessID:
		return Success, nil

	case packet.LoginCompressionID:
		return Compression, nil

	case packet.LoginPluginID:
		return Plugin, nil
	}

	return Unknown, fmt.Errorf("unfamiliar packet id: %d", id)
}

func parseErrorResponse(res string) (string, error) {
	if res == "" {
		return Empty, nil
	}

	if res == "\"Connection throttled! Please wait before reconnecting.\"" {
		return Unknown, ConnectionThrottled
	}

	versionMismatch := regexp.MustCompile("^\"Outdated client! Please use \\d\\.\\d+\\.\\d+\"$")
	if versionMismatch.MatchString(res) {
		return Unknown, fmt.Errorf("version mismatch: %s", res)
	}

	// Forge disconnect message:
	// This server has mods that require Forge to be installed on the client. \
	// Contact your server admin for more details.
	// or
	// This server has mods that require FML/Forge to be installed on the client. [...]
	if strings.Contains(res, "Forge") {
		return Forge, nil
	}

	return Unknown, nil
}

type DisconnectMsg struct {
	Translate string   `json:"translate"`
	With      []string `json:"with"`
	Text      string   `json:"text"`
}

func NewDisconnectMsg(res string) (*DisconnectMsg, error) {
	msg := new(DisconnectMsg)
	if err := json.Unmarshal([]byte(res), msg); err != nil {
		return nil, fmt.Errorf("failed to parse disconnect message: %w", err)
	}

	return msg, nil
}

// Fingerprint tries to determine the underlying software of a Minecraft server by
// analyzing the returned bad login packet disconnect message.
// Heavily inspired by matscan:
// https://github.com/mat-1/matscan/blob/master/src/processing/minecraft_fingerprinting.rs
func (m *DisconnectMsg) Fingerprint() (string, error) {
	if m.Text == "This server is only compatible with Minecraft 1.13 and above." {
		return Velocity, nil
	}

	if m.Text == "Connection throttled! Please wait before reconnecting." {
		return Unknown, ConnectionThrottled
	}

	if m.Translate == "" {
		return Unknown, errors.New("empty error topic")
	}

	if m.Translate != "disconnect.genericReason" && m.Translate != "%s" {
		return Unknown, fmt.Errorf("server responded with unfamiliar error topic: %s", m.Translate)
	}

	if len(m.With) < 1 {
		return Unknown, errors.New("incomplete disconnect message")
	}

	// example disconnect message (Spigot 1.20.4 / 765)
	// {
	//	"translate": "disconnect.genericReason",
	//	"with": [
	//		"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: \
	//		Packet login/0 (PacketLoginInStart) was larger than I expected, found 1 bytes extra \
	//		whilst reading packet 0"
	//		]
	//	}
	msg := strings.TrimPrefix(
		m.With[0],
		"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: Packet ")

	msg = regexp.MustCompile(
		" was larger than I expected, found (?:\\d+|(?:login/)?serverbound/minecraft:hello)"+
			" bytes extra whilst reading packet (?:\\d+|serverbound/minecraft:hello)$").ReplaceAllString(msg, "")

	msg = regexp.MustCompile("^(login|\\d+)/(serverbound/minecraft:hello|\\d+) ").ReplaceAllString(msg, "")

	if msg == "(PacketLoginInStart)" {
		return CraftBukkit, nil
	}

	// paper or forge without mods
	if msg == "(ServerboundHelloPacket)" {
		return Paper, nil
	}

	// vanilla e.g.: login/0 (afu) or login/serverbound/minecraft:hello (aio)
	if regexp.MustCompile("^\\(.{2,3}?\\)$").MatchString(msg) {
		return Vanilla, nil
	}

	// fabric e.g.: 2/0 (class_2915)
	if regexp.MustCompile("^\\(class_\\d*\\)$").MatchString(msg) {
		return Fabric, nil
	}

	return Unknown, nil
}

func (m *DisconnectMsg) VersionMismatch() (bool, string) {
	switch m.Translate {
	case "multiplayer.disconnect.incompatible":
		fallthrough

	case "multiplayer.disconnect.outdated_client":
		if len(m.With) < 1 {
			return true, ""
		}
		return true, m.With[0]
	}

	return false, ""
}
