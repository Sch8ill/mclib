# mclib

[![Release](https://img.shields.io/github/release/sch8ill/mclib.svg?style=flat-square)](https://github.com/sch8ill/mclib/releases)
[![doc](https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs)](https://pkg.go.dev/github.com/sch8ill/mclib)
[![Go Report Card](https://goreportcard.com/badge/github.com/sch8ill/mclib)](https://goreportcard.com/report/github.com/sch8ill/mclib)
![MIT license](https://img.shields.io/badge/license-MIT-green)

---

The `mclib` package provides utilities for interacting with Minecraft servers using the Server List Ping (SLP) protocol.
It includes functionality to query Minecraft servers for status and latency information.

## Installation

To use this package in your Go project, simply install it:

```bash
go get github.com/sch8ill/mclib
```

## Usage

### MCServer

`MCServer` represents a Minecraft server with its address and client. It provides methods to retrieve server status and
perform a status ping.

#### Creating an MCServer Instance

```go
package main

import (
	"github.com/sch8ill/mclib/server"
)

func main() {
	srv, err := server.New("example.com:25565")
	if err != nil {
		// handle error
	}
}
```

#### StatusPing

```go
res, err := srv.StatusPing()
if err != nil {
// handle error
}

fmt.Printf("version: %s\n", res.Version.Name)
fmt.Printf("protocol: %d\n", res.Version.Protocol)
fmt.Printf("online players: %d\n", res.Players.Online)
fmt.Printf("max players: %d\n", res.Players.Max)
fmt.Printf("sample players: %+q\n", res.Players.Sample)
fmt.Printf("description: %s\n", res.Description.String())
fmt.Printf("latency: %dms\n", res.Latency)
// ... 
```

#### Ping

```go
latency, err := srv.ping()
if err != nil {
// handle error
}

fmt.Printf("latency: %dms\n", latency)
```

### Cli

#### Build

requires:

```
make
go >= 1.20
```

build:

```bash
make build && mv build/mcli mcli
```

#### Usage

`mclib` also provides a simple command line interface:

```
  -addr string
        the server address (default "localhost")
  -srv
        whether a srv lookup should be made (default true)
  -timeout duration
        the connection timeout (default 5s)
```

For example:

```bash
mcli --addr hypixel.net --timeout 10s
```

## License

This package is licensed under the [MIT License](LICENSE).

---
