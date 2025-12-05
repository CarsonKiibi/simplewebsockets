package simplewebsockets

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Close state enum
type CloseState int

const (
	StateOpen CloseState = iota
	StateClosing  // we initiated close, waiting for response
	StateClosed   // close handshake complete
)

// Represents a single connection between a server and a client.
type Connection struct {
	conn    net.Conn
	writeMx sync.Mutex

	OnMessage func([]byte)
	OnClose   func([]byte)

	readBuf    []byte
	writeBuf   []byte
	maxSize    int64
	maxFrameSize int64

	frameBuffer []byte // Accumulates bytes until we have complete frames

	// close state tracking
	closeState  CloseState
	closeMx     sync.Mutex
	closeReason []byte
}

// Represents a websockets server and manages its attributes and events.
type Server struct {
	connections   map[*Connection]bool
	connectionsMx sync.RWMutex

	maxMessageSize int64
	maxFrameSize   int64

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

// OnError is called on an unclean disconnect
func (s *Server) OnError(fn func(*Connection, error)) {
	s.onError = fn
}

// Creates a new server with options. Default values are maxMessageSize = 32 kb, maxFrameSize = 16kb, readTimeout = 120 seconds, writeTimeout = 10 seconds.
// Large message/frame sizes may put the application at higher risk of Denial-of-Service attacks.
func NewServer(options ...ServerOption) *Server {
	s := &Server{
		connections:       make(map[*Connection]bool),
		maxMessageSize:    32 * 1024, // 32 kb
		maxFrameSize:      16 * 1024, // 16 kb
		handeshakeTimeout: 30 * time.Second,
		readTimeout:       120 * time.Second,
		writeTimeout:      10 * time.Second,
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
func WithMaxFrameSize(size int64) ServerOption {
    return func(s *Server) {
        s.maxFrameSize = size
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

// Helper function to determine how many bytes a complete frame should be
// Returns -1 if we don't have enough bytes to determine frame size yet
// Returns -2 if frame is too large
func frameSize(data []byte, maxFrameSize int64) int {
	if len(data) < 2 {
		return -1 // need at least 2 bytes for header
	}

	payloadLen := int64(data[1] & 0x7F)
	masked := (data[1] & 0x80) != 0

	headerSize := 2

	// add extended payload length bytes if needed
	if payloadLen == 126 {
		if len(data) < 4 {
			return -1 // need 4 bytes total for 16-bit length
		}
		headerSize = 4
		payloadLen = int64(binary.BigEndian.Uint16(data[2:4]))
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return -1 // need 10 bytes total for 64-bit length
		}
		headerSize = 10
		payloadLen = int64(binary.BigEndian.Uint64(data[2:10]))
	}

	// add mask key bytes
	if masked {
		headerSize += 4
	}

	totalFrameSize := int64(headerSize) + payloadLen

	// validate against maxFrameSize
	if totalFrameSize > maxFrameSize {
		return -2 // frame too large
	}

	return int(totalFrameSize)
}

// handles frames based on opcode
func (s *Server) processFrame(c *Connection, fr *Frame, msg *[]byte) error {
	switch fr.Opcode {
	case 0x0: // continue
		if len(*msg) == 0 {
			c.Close(1002, "Unexpected continuation frame")
			return fmt.Errorf("continuation frame without initial frame")
		}
		*msg = append(*msg, fr.Payload...)

	case 0x1: // text frame
		if len(*msg) > 0 {
			c.Close(1002, "Unexpected text frame")
			return fmt.Errorf("text frame while message in progress")
		}
		*msg = append(*msg, fr.Payload...)

	case 0x2: // binary frame
		if len(*msg) > 0 {
			c.Close(1002, "Unexpected binary frame")
			return fmt.Errorf("binary frame while message in progress")
		}
		*msg = append(*msg, fr.Payload...)

	case 0x8: // close frame
		return s.handleCloseFrame(c, fr)

	case 0x9: // ping
		c.SendPong(fr.Payload)

	case 0xA: // pong
		// Handle pong if needed

	default:
		c.Close(1002, "Unknown opcode")
		return fmt.Errorf("unknown opcode: %d", fr.Opcode)
	}

	// is message complete
	if fr.FIN && (fr.Opcode == 0x1 || fr.Opcode == 0x2 || fr.Opcode == 0x0) {
		if c.OnMessage != nil {
			c.OnMessage(*msg)
		}
		*msg = (*msg)[:0] // reset message buffer
	}

	return nil
}

// Handle close frame processing
func (s *Server) handleCloseFrame(c *Connection, fr *Frame) error {
	c.closeMx.Lock()
	currentState := c.closeState
	c.closeReason = fr.Payload

	if currentState == StateOpen {
		// client initiated close
		c.closeState = StateClosed
		c.closeMx.Unlock()

		responseFrame, err := NewCloseFrame([2]byte{}, "")
		if err != nil {
			return err
		}

		// echo back status code if we have it
		if len(fr.Payload) >= 2 {
			var statusBytes [2]byte
			copy(statusBytes[:], fr.Payload[:2])
			responseFrame, err = NewCloseFrame(statusBytes, "")
			if err != nil {
				return err
			}
		}

		c.writeMx.Lock()
		c.conn.Write(responseFrame.FrameToBytes())
		c.writeMx.Unlock()

		// call onClose
		if c.OnClose != nil {
			c.OnClose(fr.Payload)
		}

		// call onDisconnect for clean close
		if s.onDisconnect != nil {
			s.onDisconnect(c)
		}

	} else if currentState == StateClosing {
		// server initiated close and client responded -> clean close
		c.closeState = StateClosed
		c.closeMx.Unlock()

		// call onClose callback
		if c.OnClose != nil {
			c.OnClose(fr.Payload)
		}

		// call onDisconnect for clean close
		if s.onDisconnect != nil {
			s.onDisconnect(c)
		}
	} else {
		// already closed
		c.closeMx.Unlock()
	}

	// remove from connections and close TCP
	s.connectionsMx.Lock()
	delete(s.connections, c)
	s.connectionsMx.Unlock()
	c.conn.Close()
	return fmt.Errorf("connection closed") // Signal to stop processing
}

// handles a connection to the server
func (s *Server) handleConnection(c *Connection) {
	msg := make([]byte, 0)
	c.frameBuffer = make([]byte, 0, 4096) // start with 4kb buffer

	for {
		n, err := c.conn.Read(c.readBuf)
		if err != nil {
			c.closeMx.Lock()
			if c.closeState == StateOpen && s.onError != nil {
				s.onError(c, err)
			}
			c.closeState = StateClosed
			c.closeMx.Unlock()

			s.connectionsMx.Lock()
			delete(s.connections, c)
			s.connectionsMx.Unlock()
			c.conn.Close()
			return
		}

		c.frameBuffer = append(c.frameBuffer, c.readBuf[:n]...)

		// process all complete frames in buffer
		for {
			if len(c.frameBuffer) == 0 {
				break // no more data left to process
			}

			// check if we have completed frame
			completeFrameSize := frameSize(c.frameBuffer, c.maxFrameSize)

			if completeFrameSize == -1 {
				break 
			}

			if completeFrameSize == -2 {
				// frame too large
				c.Close(1009, "Frame too large")
				s.removeConnection(c)
				return
			}

			if len(c.frameBuffer) < completeFrameSize {
				break // dont have complete frame yet
			}

			// extract complete frame
			frameData := c.frameBuffer[:completeFrameSize]

			// get frame from bytes
			fr, err := BytesToFrame(frameData)
			if err != nil {
				if s.onError != nil {
					s.onError(c, err)
				}
				c.Close(1002, "Protocol error")
				s.removeConnection(c)
				return
			}

			// remove processed frame from buffer
			c.frameBuffer = c.frameBuffer[completeFrameSize:]

			// process frame
			if err := s.processFrame(c, fr, &msg); err != nil {
				// error handled in processFrame
				return
			}
		}
	}
}

// Server-initiated close of a connection
func (c *Connection) Close(status uint16, reason string) error {
	c.closeMx.Lock()
	defer c.closeMx.Unlock()

	if c.closeState != StateOpen {
		return fmt.Errorf("connection already closing or closed")
	}

	// convert status to [2]byte
	statusBytes := [2]byte{byte(status >> 8), byte(status & 0xFF)}
	closeFrame, err := NewCloseFrame(statusBytes, reason)
	if err != nil {
		return err
	}

	c.closeState = StateClosing

	c.writeMx.Lock()
	_, err = c.conn.Write(closeFrame.FrameToBytes())
	c.writeMx.Unlock()

	if err != nil {
		c.closeState = StateClosed
		c.conn.Close()
		return err
	}

	// close timeout (5 seconds)
	go func() {
		time.Sleep(5 * time.Second)
		c.closeMx.Lock()
		if c.closeState == StateClosing {
			c.closeState = StateClosed
			c.conn.Close()
		}
		c.closeMx.Unlock()
	}()

	return nil
}

// removes a connection from the server connections map and closes it
func (s *Server) removeConnection(c *Connection) {
    s.connectionsMx.Lock()
    delete(s.connections, c)
    s.connectionsMx.Unlock()
    c.conn.Close()
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
	lines := strings.SplitSeq(string(data), "\r\n")

	for line := range lines {
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
			conn:         conn,
			maxSize:      s.maxMessageSize,
			maxFrameSize: s.maxFrameSize,
			readBuf:      make([]byte, 1024),
			writeBuf:     make([]byte, 1024),
			closeState:   StateOpen, // initialize closed state
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

// Helper function to check if the connection is open
func (c *Connection) IsOpen() bool {
	c.closeMx.Lock()
	defer c.closeMx.Unlock()
	return c.closeState == StateOpen
}

// Returns current number of connections
func (s *Server) GetConnectionCount() int {
    s.connectionsMx.RLock()
    defer s.connectionsMx.RUnlock()
    return len(s.connections)
}

// Send a ping with a body
func (c *Connection) SendPing(body []byte) error {
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	f, err := NewPingFrame(body)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(f.FrameToBytes())
	return err
}

// Send a pong with a body (a pong body must be the same as the ping body it received)
func (c *Connection) SendPong(body []byte) error {
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	f, err := NewPongFrame(body)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(f.FrameToBytes())
	return err
}

// Does a buffered write to the connection with frames.
func (c *Connection) bufferedWrite(frames []Frame) error {
	var buf bytes.Buffer
	for _, frame := range frames {
		buf.Write(frame.FrameToBytes())
	}
	_, err := c.conn.Write(buf.Bytes())
	return err
}

// Does a streamed write to the connection with frames.
func (c *Connection) streamedWrite(frames []Frame) error {
	for _, frame := range frames {
		if _, err := c.conn.Write(frame.FrameToBytes()); err != nil {
			return err
		}
	}
	return nil
}

// Sends a binary message with the specified frame size. All frames of the message are first written to a buffer,
// then sent in a single TCP write to the connection. Also see "SendBinaryMessageStreamed"
func (c *Connection) SendBinaryMessageBuffered(msg []byte, fs int) error {
	frames := msgToFrames(msg, fs)
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	return c.bufferedWrite(frames)
}

// Sends a binary message with the specified frame size. Each frame is sent as a seperate write to the connection.
// Typically better for very large messages where we don't want to buffer the whole message first. Also see "SendBinaryMessageBuffered"
func (c *Connection) SendBinaryMessageStreamed(msg []byte, fs int) error {
	frames := msgToFrames(msg, fs)
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	return c.streamedWrite(frames)
}

// Sends a text message with the specified frame size. All frames of the message are first written to a buffer,
// then sent in a single TCP write to the connection. Also see "SendTextMessageStreamed"
func (c *Connection) SendTextMessageBuffered(msg string, fs int) error {
	frames := msgToFrames(msg, fs)
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	return c.bufferedWrite(frames)
}

// Sends a text message with the specified frame size. Each frame is sent as a seperate write to the connection.
// Typically better for very large messages where we don't want to buffer the whole message first. Also see "SendBinaryMessageBuffered"
func (c *Connection) SendTextMessageStreamed(msg string, fs int) error {
	frames := msgToFrames(msg, fs)
	c.writeMx.Lock()
	defer c.writeMx.Unlock()
	return c.streamedWrite(frames)
}