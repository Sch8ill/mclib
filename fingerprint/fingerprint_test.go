package fingerprint

import (
	"errors"
	"testing"
)

func TestNewDisconnectMsg(t *testing.T) {
	res := "{\"translate\": \"disconnect.genericReason\",\"with\": [" +
		"\"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: " +
		"Packet login/0 (PacketLoginInStart) was larger than I expected, " +
		"found 1 bytes extra whilst reading packet 0\"]}"

	_, err := NewDisconnectMsg(res)
	if err != nil {
		t.Error(err)
	}
}

func TestDisconnectMsg_Fingerprint(t *testing.T) {
	var tests = []struct {
		msg         *DisconnectMsg
		fingerprint string
		err         error
	}{
		{&DisconnectMsg{Translate: "disconnect.genericReason", With: []string{
			"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: " +
				"Packet login/serverbound/minecraft:hello (aiy) was larger than I expected, " +
				"found 1 bytes extra whilst reading packet serverbound/minecraft:hello"}},
			Vanilla, nil},
		{&DisconnectMsg{Translate: "disconnect.genericReason", With: []string{
			"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: " +
				"Packet login/serverbound/minecraft:hello (PacketLoginInStart) was larger than I expected, " +
				"found 1 bytes extra whilst reading packet serverbound/minecraft:hello"}},
			CraftBukkit, nil},
		{&DisconnectMsg{Translate: "disconnect.genericReason", With: []string{
			"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: " +
				"Packet login/serverbound/minecraft:hello (ServerboundHelloPacket) was larger than I expected, " +
				"found 1 bytes extra whilst reading packet serverbound/minecraft:hello"}},
			Paper, nil},
		{&DisconnectMsg{Translate: "disconnect.genericReason", With: []string{
			"Internal Exception: io.netty.handler.codec.DecoderException: java.io.IOException: " +
				"Packet login/serverbound/minecraft:hello (class_2915) was larger than I expected, " +
				"found 1 bytes extra whilst reading packet serverbound/minecraft:hello"}},
			Fabric, nil},
		{&DisconnectMsg{Text: "This server is only compatible with Minecraft 1.13 and above."},
			Velocity, nil},
		{&DisconnectMsg{Text: "Connection throttled! Please wait before reconnecting."},
			Unknown, ConnectionThrottled},
	}

	for _, test := range tests {
		t.Run(test.fingerprint, func(t *testing.T) {
			fingerprint, err := test.msg.Fingerprint()
			if !errors.Is(err, test.err) {
				t.Error(err)
			}

			if fingerprint != test.fingerprint {
				t.Errorf("wrong fingerprint: got %s, expected %s", fingerprint, test.fingerprint)
			}
		})
	}
}

func TestDisconnectMsg_VersionMismatch(t *testing.T) {
	var tests = []struct {
		msg      *DisconnectMsg
		mismatch bool
		version  string
	}{
		{&DisconnectMsg{Translate: "multiplayer.disconnect.incompatible", With: []string{"1.20.2"}}, true, "1.20.2"},
		{&DisconnectMsg{Translate: "multiplayer.disconnect.outdated_client", With: []string{"1.20.2"}}, true, "1.20.2"},
		{&DisconnectMsg{Translate: "disconnect.genericReason"}, false, ""},
	}

	for _, test := range tests {
		t.Run(test.msg.Translate, func(t *testing.T) {
			mismatch, version := test.msg.VersionMismatch()
			if mismatch != test.mismatch {
				t.Errorf("mismatch: got %t, want %t", mismatch, test.mismatch)
			}
			if version != test.version {
				t.Errorf("wrong version detcted: got %s, expected %s", test.version, version)
			}
		})
	}
}
