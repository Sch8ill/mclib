package main

import (
	"flag"
	"fmt"

	"github.com/sch8ill/mclib"
	"github.com/sch8ill/mclib/fingerprint"
)

func main() {
	addr := flag.String("addr", "localhost", "the server address")
	timeout := flag.Duration("timeout", mclib.DefaultTimeout, "the connection timeout")
	srv := flag.Bool("srv", true, "whether a srv lookup should be made")
	protocol := flag.Int("protocol", 760, "the protocol version number the client should use")
	doFingerprint := flag.Bool("fingerprint", true, "whether a software fingerprint should be performed on the server")
	flag.Parse()

	opts := []mclib.ClientOption{mclib.WithTimeout(*timeout), mclib.WithProtocolVersion(int32(*protocol))}
	if !*srv {
		opts = append(opts, mclib.WithoutSRV())
	}

	mcs, err := mclib.NewClient(*addr, opts...)
	if err != nil {
		panic(err)
	}

	res, err := mcs.StatusPing()
	if err != nil {
		panic(err)
	}

	fmt.Printf("version: %s\n", res.Version.Name)
	fmt.Printf("protocol: %d\n", res.Version.Protocol)
	fmt.Printf("description: %s\n", res.Description.String())
	fmt.Printf("online players: %d\n", res.Players.Online)
	fmt.Printf("max players: %d\n", res.Players.Max)
	fmt.Printf("sample players: %+q\n", res.Players.Sample)
	fmt.Printf("latency: %dms\n", res.Latency)
	fmt.Printf("favicon: %t\n", res.Favicon != "")

	if *doFingerprint {
		software, err := fingerprint.FingerprintWithProtocol(*addr, res.Version.Protocol, opts...)
		if err != nil {
			fmt.Printf("failed to perform fingerprint: %s\n", err)
		} else {
			fmt.Printf("software fingerprint: %s\n", software)
		}
	}
}
