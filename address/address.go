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
	host    string
	port    uint16
	srvHost string
	srvPort uint16
	srv     bool
	portSet bool
}

// New creates a new Address from a given address string,
// which can include the host and port separated by a colon (e.g., "example.com:25565").
// If no port is specified, it uses the default Minecraft port.
func New(addr string) (*Address, error) {
	if !strings.Contains(addr, ":") {
		return &Address{
			host: addr,
			port: DefaultPort,
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
		host:    splitAddr[0],
		port:    uint16(port),
		portSet: true,
	}, nil
}

// ResolveSRV resolves the SRV record for the Address's domain and updates its SRV fields.
// ResolveSRV does not resolve the SRV record if a port has already been set.
func (a *Address) ResolveSRV() error {
	if a.IsIP() {
		return nil
	}

	// the Notchian client does not resolve srv records when a port has already been set
	if a.portSet {
		return nil
	}

	_, records, err := net.LookupSRV("minecraft", "tcp", a.host)
	if err != nil {
		return fmt.Errorf("failed to resolve SRV record: %w", err)
	}

	if len(records) > 0 {
		srvRecord := records[0]
		a.srvPort = srvRecord.Port
		a.srvHost, _ = strings.CutSuffix(srvRecord.Target, ".")
		a.srv = true
	}

	return nil
}

// String returns the address string based on whether SRV record resolution is enabled.
// If SRV resolution is enabled, it returns the SRV address; otherwise, the original address.
func (a *Address) String() string {
	if a.srv {
		return a.SRVAddr()
	}
	return a.OGAddr()
}

// Host returns the Host of the Address.
func (a *Address) Host() string {
	if a.srv {
		return a.srvHost
	}
	return a.host
}

// Port returns the Port of the Address.
func (a *Address) Port() uint16 {
	if a.srv {
		return a.srvPort
	}
	return a.port
}

// SRVAddr returns the address string in the format "hostname:port" based on SRV record values.
func (a *Address) SRVAddr() string {
	return fmt.Sprintf("%s:%d", a.srvHost, a.srvPort)
}

// OGAddr returns the address string in the format "hostname:port".
func (a *Address) OGAddr() string {
	return fmt.Sprintf("%s:%d", a.host, a.port)
}

// IsIP checks if the host in the Address is an IP address.
func (a *Address) IsIP() bool {
	return net.ParseIP(a.host) != nil
}
