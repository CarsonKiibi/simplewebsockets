package main

import (
	"encoding/binary"
)

// must be encoded as raw bytes before being sent over tcp
type Frame struct {
	FIN           bool // 1-bit flag
	Opcode        byte
	Mask          bool // 1-bit flag
	MaskKey       [4]byte
	Payload       []byte
	PayloadLength int64
}

type Frames struct {
	MsgFrames []Frame
}

// Turns a Frame into its raw byte representation for sending via TCP.
func (f Frame) FrameToBytes() []byte {
    
    // --- Calculate Frame Size ---

    frameSize := 2 // header will always be present
    
    // determine extra bytes for payload length
    var payloadLenBytes int
    if f.PayloadLength < 126 {
        payloadLenBytes = 0
    } else if f.PayloadLength <= 65535 {
        payloadLenBytes = 2
    } else {
        payloadLenBytes = 8
    }
    
    // determine if we need extra bytes for mask 
    maskBytes := 0
    if f.Mask {
        maskBytes = 4
    }
    
	// make buffer for frame contents
    frameSize += payloadLenBytes + maskBytes + int(f.PayloadLength)
    frame := make([]byte, frameSize)
    
    // set fin and opcode
    if f.FIN {
        frame[0] |= 0x80
    }
    frame[0] |= f.Opcode
    
    // set mask bit
    if f.Mask {
        frame[1] |= 0x80
    }
    
    // set payload length bits
    if f.PayloadLength < 126 {
        frame[1] |= byte(f.PayloadLength)
        payloadOffset := 2
        
        // add mask key if present
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // copy payload
        copy(frame[payloadOffset:], f.Payload)
    } else if f.PayloadLength <= 65535 {
        frame[1] |= 126
        
        // add extended 16-bit length
        frame[2] = byte(f.PayloadLength >> 8)
        frame[3] = byte(f.PayloadLength)
        payloadOffset := 4
        
        // add mask key if exists
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // copy payload
        copy(frame[payloadOffset:], f.Payload)
    } else {
        frame[1] |= 127
        
        // add extended 64-bit length
        binary.BigEndian.PutUint64(frame[2:10], uint64(f.PayloadLength))
        payloadOffset := 10
        
        // add mask key if exists
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // copy payload
        copy(frame[payloadOffset:], f.Payload)
    }
    
    return frame
}

// Turns payload (byte array) into frames
func PayloadToFrame(payload []byte) Frame {
    frame := Frame{}

    // check if FIN 
    if payload[0] & 0x80 == 1 {
        frame.FIN = true
    }

    // set opcode
    frame.Opcode = payload[0] & 0x0F

	return frame
}