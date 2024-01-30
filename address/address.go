// Package address provides utilities for working with Minecraft server addresses.
package address

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const DefaultPort uint16 = 25565

// Address represents a Minecraft server address with a host, port and srv record.
type Address struct {
	Host    string
	Port    uint16
	SRVHost string
	SRVPort uint16
	SRV     bool
}

// New creates a new Address from a given address string,
// which can include the host and port separated by a colon (e.g., "example.com:25565").
// If no port is specified, it uses the default Minecraft port.
func New(addr string) (*Address, error) {
	if !strings.Contains(addr, ":") {
		return &Address{
			Host: addr,
			Port: DefaultPort,
		}, nil
	}

	splitAddr := strings.Split(addr, ":")
	if len(splitAddr) != 2 {
		return nil, fmt.Errorf("invalid address: %s", addr)
	}

	port, err := strconv.Atoi(splitAddr[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", splitAddr[1])
	}

	return &Address{
		Host: splitAddr[0],
		Port: uint16(port),
	}, nil
}

// ResolveSRV resolves the SRV record for the Address's host and updates its SRV fields.
func (a *Address) ResolveSRV() error {
	if a.IsIP() {
		return nil
	}

	_, records, err := net.LookupSRV("minecraft", "tcp", a.Host)
	if err != nil {
		return fmt.Errorf("failed to resolve SRV record: %w", err)
	}

	if len(records) > 0 {
		srvRecord := records[0]
		a.SRVPort = srvRecord.Port
		a.SRVHost, _ = strings.CutSuffix(srvRecord.Target, ".")
		a.SRV = true
	}

	return nil
}

// IsIP checks if the host in the Address is an IP address.
func (a *Address) IsIP() bool {
	return net.ParseIP(a.Host) != nil
}

// Addr returns the address string based on whether SRV record resolution is enabled.
// If SRV resolution is enabled, it returns the SRV address; otherwise, the original address.
func (a *Address) Addr() string {
	if a.SRV {
		return a.SRVAddr()
	}
	return a.OGAddr()
}

// SRVAddr returns the address string in the format "hostname:port" based on SRV record values.
func (a *Address) SRVAddr() string {
	return fmt.Sprintf("%s:%d", a.SRVHost, a.SRVPort)
}

// OGAddr returns the address string in the format "hostname:port".
func (a *Address) OGAddr() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}
