package server

import (
	"log"
	"net"
	"time"
)

var DefaultServer Server = Server{
	timeout: time.Second * 15,
}

type Server struct {
	timeout time.Duration
}

func (s *Server) Listen(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		handler := NewHandler(conn, s.timeout)
		go func() {
			if err := handler.Handle(); err != nil {
				log.Printf("%s: %s", handler.Address.String(), err.Error())
			}
		}()
	}
}
