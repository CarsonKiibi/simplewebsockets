package main

import "fmt"

func main() {
	myServer := NewServer()

	//onConnect handler
	myServer.OnConnect(func(c *Connection) {
		fmt.Println("Client connected")
		
		// onmessage handler
		c.onMessage = func(data []byte) {
			fmt.Printf("Received message: %s\n", string(data))
			
			// need to implement sending
			c.SendTextMessageBuffered(string(data), 4)
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

	fmt.Println("Starting WebSocket server...")
	err := myServer.Listen("localhost:8080")
	if err != nil {
		panic(err)
	}
}
