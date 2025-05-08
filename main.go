package main

import (
	"fmt"
	//"github.com/carsonkiibi/websockets/server"
)

func main() {
	// server.PrintBits()
	// number := 0
	// anded := number | 0b1111
	// fmt.Println(anded)

	frameSize := 2 // at least 2 for header

	currFrame := make([]byte, frameSize)
		
	if (true) {
		currFrame[0] |= 0x80
	}

	currFrame[0] |= 0xf
	fmt.Printf("%b", currFrame)
}
