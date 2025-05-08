package server

import (
	"net"
	"sync"
	"time"
)

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

type Server struct {
	connections map[*Connection]bool
	connectionsMx sync.RWMutex

	maxMessageSize int64 
	readTimeout time.Duration
	writeTimeout time.Duration

	onConnect func(*Connection)
	onDisconnect func(*Connection)
	onError func(*Connection, error)
}

func NewServer() *Server {
	return NewServer()
}