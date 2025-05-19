package main

import (
	"net"
	"sync"
	"time"
)

// Represents a single connection between a server and a client.
type Connection struct {
	conn net.Conn 
	writeMx sync.Mutex

	closed bool 

	onMessage func([]byte)
	onClose func()

	readBuf []byte 
	writeBuf []byte 
	maxSize int64
}

// Represents a websockets server and manages its attributes and events.
type Server struct {
	connections map[*Connection]bool
	connectionsMx sync.RWMutex

	maxMessageSize int64 

	handeshakeTimeout time.Duration
	readTimeout time.Duration
	writeTimeout time.Duration

	onConnect func(*Connection)
	onDisconnect func(*Connection)
	onError func(*Connection, error)
}

type ServerOption func(*Server)

// Creates a new server with options. Default values are maxMessageSize = 32 kb, readTimeout = 120 seconds, writeTimeout = 10 seconds.
func NewServer(options ...ServerOption) *Server {
	s := &Server{
		connections: make(map[*Connection]bool),
		maxMessageSize: 32*1024, // 32 kb
		readTimeout: 120 * time.Second,
		writeTimeout: 10 * time.Second,
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



