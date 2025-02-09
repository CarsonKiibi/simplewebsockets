package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// must be encoded as raw bytes before being sent over tcp
type Frame struct {
	FIN bool // 1-bit flag
	Opcode byte
	Mask bool // 1-bit flag
	MaskKey [4]byte
	Payload []byte
}

type Frames struct {
	MsgFrames []Frame
}

func (fs Frames) FrameToPayload() []byte {
	var buf bytes.Buffer
	for f := range fs.MsgFrames {
		if (f.FIN == true) {

		}
	}
} 

// Turns payload (byte array) into frames
func PayloadToFrames([]byte) []Frame {


	return []Frame{}
}
``
func PrintBits() {

}

