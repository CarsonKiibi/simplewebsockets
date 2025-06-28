package main

import (
	"fmt"
	"time"
)

func main() {
	myServer := NewServer(WithMaxMessageSize(100*1024))

	//onConnect handler
	myServer.OnConnect(func(c *Connection) {
		fmt.Println("Client connected")

		// onmessage handler
		c.onMessage = func(data []byte) {
			fmt.Printf("Received message: %s\n", string(data))

			// need to implement sending
			c.SendBinaryMessageBuffered([]byte{0x12, 0x34}, 4)
		}

		// close handler
		c.onClose = func(reason []byte) {
			fmt.Printf("Closed for reason: %s\n", string(reason))
		}
	})

	myServer.OnDisconnect(func(c *Connection) {
		fmt.Println("Client disconnected cleanly")
	})

	myServer.OnError(func(c *Connection, err error) {
		fmt.Printf("Connection error: %v\n", err)
	})

	// Start connection counter goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			connectionCount := myServer.GetConnectionCount()
			fmt.Printf("[STATUS] Active connections: %d\n", connectionCount)
		}
	}()

	fmt.Println("Starting WebSocket server...")
	err := myServer.Listen("localhost:8080")
	if err != nil {
		panic(err)
	}
}
