package main

import (
	"fmt"
)

func main() {
	// server.PrintBits()
	// number := 0
	// anded := number | 0b1111
	// fmt.Println(anded)

	myFrame := Frame{
		FIN: true,
		Opcode: 0x7,
		Mask: true,
		MaskKey: [4]byte{0x1, 0x2, 0x1, 0x2},
		Payload: []byte{0x46, 0x65, 0x6c, 0x6c, 0x6f},
		PayloadLength: 5,
	}
		
	bytes := myFrame.FrameToBytes()
	fmt.Printf("%b", bytes)
}
