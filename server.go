package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Represents a single connection between a server and a client.
type Connection struct {
	conn    net.Conn
	writeMx sync.Mutex

	closed bool

	onMessage func([]byte)
	onClose   func()

	readBuf  []byte
	writeBuf []byte
	maxSize  int64
}

// Represents a websockets server and manages its attributes and events.
type Server struct {
	connections   map[*Connection]bool
	connectionsMx sync.RWMutex

	maxMessageSize int64

	handeshakeTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration

	onConnect    func(*Connection)
	onDisconnect func(*Connection)
	onError      func(*Connection, error)
}

type ServerOption func(*Server)

// Creates a new server with options. Default values are maxMessageSize = 32 kb, readTimeout = 120 seconds, writeTimeout = 10 seconds.
func NewServer(options ...ServerOption) *Server {
	s := &Server{
		connections:    make(map[*Connection]bool),
		maxMessageSize: 32 * 1024, // 32 kb
		readTimeout:    120 * time.Second,
		writeTimeout:   10 * time.Second,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// Setter to be passed into the creation of a server.
func WithMaxMessageSize(size int64) ServerOption {
	return func(s *Server) {
		s.maxMessageSize = size
	}
}

// Setter to be passed into the creation of a server.
func WithReadTimeout(seconds uint16) ServerOption {
	return func(s *Server) {
		s.readTimeout = time.Duration(seconds) * time.Second
	}
}

// Setter to be passed into the creation of a server.
func WithWriteTimeout(seconds uint16) ServerOption {
	return func(s *Server) {
		s.writeTimeout = time.Duration(seconds) * time.Second
	}
}

func (s *Server) handleConnection(c *Connection) {
	for {
		len, err := c.conn.Read(c.readBuf)
		if err != nil {
			return
		}

		fmt.Printf("Message received: %s\n", string(c.readBuf[:len]))
	}
}

func (s *Server) performHandshake(c net.Conn) error {
	fmt.Print("Handshake simulation\n")
	return nil
}

// Starts listening for a server, and accepts incoming connections.
func (s *Server) Listen(address string) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	// begin connection loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.onError != nil {
				s.onError(nil, err)
			}
			continue
		}

		// set the handshake timeout
		conn.SetDeadline(time.Now().Add(s.handeshakeTimeout))

		if err := s.performHandshake(conn); err != nil {
			conn.Close()
			if s.onError != nil {
				s.onError(nil, err)
			}
		}

		conn.SetDeadline(time.Time{})

		c := &Connection{
			conn:     conn,
			maxSize:  s.maxMessageSize,
			readBuf:  make([]byte, 1024),
			writeBuf: make([]byte, 1024),
		}

		s.connectionsMx.Lock()
		s.connections[c] = true
		s.connectionsMx.Unlock()

		if s.onConnect != nil {
			s.onConnect(c)
		}

		go s.handleConnection(c)
	}
}
