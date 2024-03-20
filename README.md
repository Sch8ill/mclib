# mclib

[![Release](https://img.shields.io/github/release/sch8ill/mclib.svg?style=flat-square)](https://github.com/sch8ill/mclib/releases)
[![doc](https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs)](https://pkg.go.dev/github.com/sch8ill/mclib)
[![Go Report Card](https://goreportcard.com/badge/github.com/sch8ill/mclib)](https://goreportcard.com/report/github.com/sch8ill/mclib)
![MIT license](https://img.shields.io/badge/license-MIT-green)

---

The `mclib` package provides utilities for interacting with Minecraft servers using
the [Minecraft protocol](https://wiki.vg/Protocol).
It includes functionality to query Minecraft servers for status and latency information.
`mclib` is also capable of determining the software a server is running on by using fingerprinting techniques.

---

## Installation

To use this package in your Go project, simply install it:

```bash
go get github.com/sch8ill/mclib
```

## Usage

### StatusPing

```go
package main

import (
	"fmt"

	"github.com/sch8ill/mclib"
)

func main() {
	client, _ := mclib.NewClient("2b2t.org")
	res, _ := client.StatusPing()

	fmt.Printf("version: %s\n", res.Version.Name)
	fmt.Printf("protocol: %d\n", res.Version.Protocol)
	fmt.Printf("online players: %d\n", res.Players.Online)
	fmt.Printf("max players: %d\n", res.Players.Max)
	fmt.Printf("sample players: %+q\n", res.Players.Sample)
	fmt.Printf("description: %s\n", res.Description.String())
	fmt.Printf("latency: %dms\n", res.Latency)
}
```

#### output

```text
version: Velocity 1.7.2-1.20.4
protocol: 47
online players: 571
max players: 1
sample players: [{"Fit" "fdee323e-7f0c-4c15-8d1c-0f277442342a"}]
description: 2B Updated to 1.19! 2T
latency: 8ms
```

### Fingerprint

```go
package main

import (
	"fmt"

	"github.com/sch8ill/mclib/fingerprint"
)

func main() {
	software, _ := fingerprint.Fingerprint("localhost")
	fmt.Printf("software fingerprint: %s\n", software)
}
```

#### output

```text
software fingerprint: craftbukkit
```

Further documentation can be found on [pkg.go.dev](https://pkg.go.dev/github.com/sch8ill/mclib).

---

### Cli

#### Build

requires:

```
make
go >= 1.22
```

```bash
make build
```

#### Usage

`mclib` also provides a simple command line interface:

```
  -addr string
        the server address (default "localhost")
  -fingerprint
        whether a software fingerprint should be performed on the server (default true)
  -protocol int
        the protocol version number the client should use (default 760)
  -srv
        whether a srv lookup should be made (default true)
  -timeout duration
        the connection timeout (default 5s)
```

For example:

```bash
mcli --addr hypixel.net --timeout 10s
```

---

## License

This package is licensed under the [MIT License](LICENSE).

---
