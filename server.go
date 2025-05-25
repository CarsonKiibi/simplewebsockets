package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
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

// OnConnect is called when a client first connects to the server succesfully (after the http handshake).
func (s *Server) OnConnect(fn func(*Connection)) {
    s.onConnect = fn
}

// OnDisconnect is called after a clean disconnect (no error has occured and the server and client have completed a closing handshake).
func (s *Server) OnDisconnect(fn func(*Connection)) {
    s.onDisconnect = fn
}

// OnError is called 
func (s *Server) OnError(fn func(*Connection, error)) {
    s.onError = fn
}

// Creates a new server with options. Default values are maxMessageSize = 32 kb, readTimeout = 120 seconds, writeTimeout = 10 seconds.
func NewServer(options ...ServerOption) *Server {
	s := &Server{
		connections:    make(map[*Connection]bool),
		maxMessageSize: 32 * 1024, // 32 kb
		handeshakeTimeout: 30 * time.Second,
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

// Handles each connection (called as go func)
// need better error handling 
func (s *Server) handleConnection(c *Connection) {
	for {
		len, err := c.conn.Read(c.readBuf)
		if err != nil {
			if s.onError != nil {
				s.onError(c, err)
			}
			s.connectionsMx.Lock()
			delete(s.connections, c)
			s.connectionsMx.Unlock()
			return
		}

		c.onMessage(c.readBuf[:len])
	}
}

func (s *Server) performServerHandshake(c net.Conn, key []byte) error {
	status := "HTTP/1.1 101 Switching Protocols"
	upgrade := "websocket"
	connection := "Upgrade"
	var guid = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

	bytes := append(key, guid...)
	hasher := sha1.New()
	hasher.Write(bytes)
	wsAccept := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	req := fmt.Sprintf("%s\r\nUpgrade: %s\r\nConnection: %s\r\nSec-WebSocket-Accept: %s\r\n\r\n", status, upgrade, connection, wsAccept)

	_, err := c.Write([]byte(req))
	if err != nil {
		fmt.Println("Error sending server handshake:", err)
		return err
	}
	return nil
}

func getWebSocketKey(data []byte) (string, error) {
	lines := strings.Split(string(data), "\r\n")
    
    for _, line := range lines {
        if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) == 2 {
                return strings.TrimSpace(parts[1]), nil
            }
        }
    }
    
    return "", fmt.Errorf("Sec-WebSocket-Key header not found")
}

// Starts listening for a server, and accepts incoming connections.
func (s *Server) Listen(address string) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	fmt.Printf("Listening on %s \n", address)

	// begin connection loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.onError != nil {
				s.onError(nil, err)
			}
			continue
		}

		fmt.Println("Connection got here")
		
		conn.SetDeadline(time.Now().Add(s.handeshakeTimeout))
		hsBuf := make([]byte, 1024)
		len, err := conn.Read(hsBuf)

		if err != nil {
			return err
		}

		req := hsBuf[:len]
		
		if !strings.HasPrefix(string(req), "GET ") {
			return fmt.Errorf("client did not send handshake (not a GET http request)")
		}

		key, err := getWebSocketKey(req)
		if err != nil {
			return err
		}

		if err := s.performServerHandshake(conn, []byte(key)); err != nil {
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

		fmt.Println("Handling new connection")
		go s.handleConnection(c)
	}
}
