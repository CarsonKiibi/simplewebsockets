package main

import (
	"encoding/binary"
	"fmt"
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
			copy(frame[payloadOffset:payloadOffset + 4], f.MaskKey[:])
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

// BytesToFrame converts raw TCP bytes into a WebSocket Frame struct.
func BytesToFrame(data []byte) (*Frame, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("frame too short: need at least 2 bytes, got %d", len(data))
	}

	frame := &Frame{}

	// parse first byte -> FIN (1 bit) + RSV (3 bit) + opcode (4 bit)
	firstByte := data[0]
	frame.FIN = (firstByte & 0x80) != 0 // check fin bit
	frame.Opcode = firstByte & 0x0F     // check opcode bits

	// parse second byte -> mask (1 bit) + payload length (7 bits)
	secondByte := data[1]
	frame.Mask = (secondByte & 0x80) != 0 // check mask bit (boolean)
	payloadLen := secondByte & 0x7F       // check len bits

	offset := 2 // variable offset past this point

	// find payload length
	switch {
	case payloadLen < 126:
		frame.PayloadLength = int64(payloadLen)

	case payloadLen == 126:
		// 2 more bytes contain length
		if len(data) < offset + 2 {
			return nil, fmt.Errorf("frame too short for 16-bit length")
		}
		frame.PayloadLength = int64(binary.BigEndian.Uint16(data[offset : offset + 2]))
		offset += 2

	case payloadLen == 127:
		// 8 more bytes contain length
		if len(data) < offset + 8 {
			return nil, fmt.Errorf("frame too short for 64-bit length")
		}
		frame.PayloadLength = int64(binary.BigEndian.Uint64(data[offset : offset + 8]))
		offset += 8
	}

	// get mask key if present
	if frame.Mask {
		if len(data) < offset+4 {
			return nil, fmt.Errorf("frame too short for mask key")
		}
		copy(frame.MaskKey[:], data[offset:offset + 4])
		offset += 4
	}

	// check if actual payload length is expected based on provided length
	if len(data) < offset+int(frame.PayloadLength) {
		return nil, fmt.Errorf("frame too short for payload: expected %d bytes, got %d",
			offset+int(frame.PayloadLength), len(data))
	}

	// extract and unmask payload if necessary
	frame.Payload = make([]byte, frame.PayloadLength)
	copy(frame.Payload, data[offset:offset + int(frame.PayloadLength)])

	if frame.Mask {
		// unmask payload
		for i := range frame.Payload {
			frame.Payload[i] ^= frame.MaskKey[i % 4]
		}
	}

	return frame, nil
}

func NewFrame[T string | []byte](opcode byte, data T, isFin bool, masked bool, maskKey [4]byte) Frame {
	var payload []byte

	switch d := any(data).(type) {
	case string:
		payload = []byte(d)
	case []byte:
		payload = d
	}

	plen := len(data)

	return Frame{
		FIN:           isFin,
		Opcode:        opcode, // change type of opcode to type Opcode
		Mask:          masked,
		MaskKey:       maskKey,
		Payload:       payload,
		PayloadLength: int64(plen),
	}
}

// Create closed frame with given status and reason. Some clients may reject a status of 0. Use NewEmptyCloseFrame to not give a status.
func NewCloseFrame(status [2]byte, reason string) (Frame, error) {
	if (len(status) + len(reason)) > 125 {
		return Frame{}, fmt.Errorf("error: control frame (close) should not have body (status + reason) larger than 125 bytes (). ")
	}
    byteReason := []byte(reason)
    body := append(status[:], byteReason...)
    return NewFrame(0x8, body, true, false, [4]byte{}), nil
}

// Create closed frame with no body
func NewEmptyCloseFrame() Frame {
    return NewFrame(0x8, []byte{}, true, false, [4]byte{})
}

func NewPingFrame(body []byte) (Frame, error) {
	if len(body) > 125 {
		return Frame{}, fmt.Errorf("error: control frame (ping) must not have a body greater than 125 bytes")
	}

	return NewFrame(0x9, body, true, false, [4]byte{}), nil
}

func NewPongFrame(body []byte) (Frame, error) {
	if len(body) > 125 {
		return Frame{}, fmt.Errorf("error: control frame (pong) must not have a body greater than 125 bytes")
	}

	return NewFrame(0xA, body, true, false, [4]byte{}), nil
}

// Converts a text or binary message 
func msgToFrames[M string | []byte](msg M, fs int) []Frame {
    if fs <= 0 {
        panic("frame size must be positive")
    }
    
    msgLen := len(msg)
    if msgLen == 0 {
        // Return single empty frame
        var opcode byte
        switch any(msg).(type) {
        case string:
            opcode = 1 // Text frame
        case []byte:
            opcode = 2 // Binary frame
        }
        
        return []Frame{{
            FIN:           true,
            Opcode:        opcode,
            Mask:          false,
            MaskKey:       [4]byte{},
            Payload:       []byte{},
            PayloadLength: 0,
        }}
    }

    frameCount := (msgLen + fs - 1) / fs // ceiling division
    frames := make([]Frame, 0, frameCount) // allocate for frames

    var opcode byte
    switch any(msg).(type) {
    case string:
        opcode = 1 // Text frame
    case []byte:
        opcode = 2 // Binary frame
    }

	// convert message to bytes for processing
    var msgBytes []byte
    switch m := any(msg).(type) {
    case string:
        msgBytes = []byte(m)
    case []byte:
        msgBytes = m
    }

    for i := 0; i < frameCount; i++ {
        start := i * fs
        end := start + fs
        if end > msgLen {
            end = msgLen
        }

        payload := msgBytes[start:end]
        isLastFrame := (i == frameCount-1)
        
        // first frame has opcode, rest use continuation frame
        frameOpcode := opcode
        if i > 0 {
            frameOpcode = 0 // Continuation frame
        }

        frame := Frame{
            FIN:           isLastFrame,
            Opcode:        frameOpcode,
            Mask:          false, //server-to-client need not be masked (maybe change to work for both tho later)
            MaskKey:       [4]byte{},
            Payload:       payload,
            PayloadLength: int64(len(payload)),
        }

        frames = append(frames, frame)
    }

    return frames
}
