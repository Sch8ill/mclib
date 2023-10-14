package main

import (
	"flag"
	"fmt"

	"github.com/sch8ill/mclib/server"
	"github.com/sch8ill/mclib/slp"
)

func main() {
	addr := flag.String("addr", "localhost", "the server address")
	timeout := flag.Duration("timeout", slp.DefaultTimeout, "the connection timeout")
	srv := flag.Bool("srv", true, "whether a srv lookup should be made")
	flag.Parse()

	opts := []server.MCServerOption{server.WithTimeout(*timeout)}
	if !*srv {
		opts = append(opts, server.WithoutSRV())
	}

	mcs, err := server.New(*addr, opts...)
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
}
