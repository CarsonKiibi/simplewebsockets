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
    
    // Set FIN and opcode
    if f.FIN {
        frame[0] |= 0x80
    }
    frame[0] |= f.Opcode
    
    // Set mask bit and payload length
    if f.Mask {
        frame[1] |= 0x80
    }
    
    // Handle payload length
    if f.PayloadLength < 126 {
        frame[1] |= byte(f.PayloadLength)
        payloadOffset := 2
        
        // Add mask key if present
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // Copy payload
        copy(frame[payloadOffset:], f.Payload)
    } else if f.PayloadLength <= 65535 {
        frame[1] |= 126
        
        // Add extended 16-bit length
        frame[2] = byte(f.PayloadLength >> 8)
        frame[3] = byte(f.PayloadLength)
        payloadOffset := 4
        
        // Add mask key if present
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // Copy payload
        copy(frame[payloadOffset:], f.Payload)
    } else {
        frame[1] |= 127
        
        // Add extended 64-bit length
        binary.BigEndian.PutUint64(frame[2:10], uint64(f.PayloadLength))
        payloadOffset := 10
        
        // Add mask key if present
        if f.Mask {
            copy(frame[payloadOffset:payloadOffset+4], f.MaskKey[:])
            payloadOffset += 4
        }
        
        // Copy payload
        copy(frame[payloadOffset:], f.Payload)
    }
    
    return frame
}

// Turns payload (byte array) into frames
func PayloadToFrames([]byte) []Frame {

	return []Frame{}
}
func PrintBits() {

}

// switch f.Opcode {
// case 0x0:
// 	fmt.Println("0x0: Continue Frame")
// case 0x1:
// 	fmt.Println("0x1: Text Frame")
// case 0x2:
// 	fmt.Println("0x2: Binary Frame")
// case 0x3:
// 	fmt.Println("0x3: Reserved Frame")
// case 0x4:
// 	fmt.Println("0x4: Reserved Frame")
// case 0x5:
// 	fmt.Println("0x5: Reserved Frame")
// case 0x6:
// 	fmt.Println("0x6: Reserved Frame")
// case 0x7:
// 	fmt.Println("0x7: Reserved Frame")
// case 0x8:
// 	fmt.Println("0x8: Connection Closed")
// case 0x9:
// 	fmt.Println("0x9: Ping")
// case 0xa:
// 	fmt.Println("0xa: Pong")
// case 0xb:
// 	fmt.Println("0xb: Reserved control frame")
// case 0xc:
// 	fmt.Println("0xc: Reserved control frame")
// case 0xd:
// 	fmt.Println("0xd: Reserved control frame")
// case 0xe:
// 	fmt.Println("0xe: Reserved control frame")
// case 0xf:
// 	fmt.Println("0xf: Reserved control frame")
// default:
// 	fmt.Println("Impossible Opcode")
// }
