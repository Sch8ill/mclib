package main

import "github.com/sch8ill/mclib/server"

func main() {
	s := &server.DefaultServer

	if err := s.Listen("0.0.0.0:25565"); err != nil {
		panic(err)
	}
}
