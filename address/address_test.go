package address

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	host := "localhost"
	var port uint16 = 25565
	raw := fmt.Sprintf("%s:%d", host, port)

	addr, err := New(raw)
	if err != nil {
		t.Fatal(err)
	}

	if addr.host != host {
		t.Errorf("host is %s, want %s", addr.Host(), host)
	}

	if addr.port != port {
		t.Errorf("port is %d, want %d", addr.Port(), port)
	}

	t.Run("invalid format", func(t *testing.T) {
		if _, err := New(""); err == nil {
			t.Errorf("New should return an error")
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		if _, err := New("localhost:-1"); err == nil {
			t.Errorf("New should return an error")
		}
	})
}
