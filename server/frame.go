package server

// "fmt"

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

func (fs Frames) FrameToBytes() []byte {
	// var buf bytes.Buffer
	for _, f := range fs.MsgFrames {

		frameSize := 2 // at least 2 for header

		if f.Mask {
			frameSize += 4 // add mask if it exists
		}

		frameSize += int(f.PayloadLength) // add payload length

		maskOffset := 0

		if (f.PayloadLength >= 126) && (f.PayloadLength <= 65535) {
			frameSize += 2
			maskOffset += 2
		} else if f.PayloadLength > 65536 {
			frameSize += 8
			maskOffset += 8 // i think this is wrong
		}

		currFrame := make([]byte, frameSize)

		if f.FIN {
			currFrame[0] |= 0x80
		}

		currFrame[0] |= f.Opcode

		if f.Mask {
			currFrame[1] |= 0x80

			currFrame[2] |= f.MaskKey[0]
			currFrame[3] |= f.MaskKey[1]
			currFrame[4] |= f.MaskKey[2]
			currFrame[5] |= f.MaskKey[3]
		}

		payloadOffset := 0

		if f.PayloadLength < 126 {
			currFrame[1] |= byte(f.PayloadLength)
		} else if (f.PayloadLength >= 126) && (f.PayloadLength <= 65535) {
			currFrame[1] |= 0xef
			if f.Mask {
				payloadOffset += 4 
			}
			currFrame[2 + payloadOffset] = byte(f.PayloadLength >> 8)
			currFrame[3 + payloadOffset] = byte(f.PayloadLength)
		} else {
			currFrame[1] |= 0xef
			if f.Mask {
				payloadOffset += 4 
			}
			currFrame[2 + payloadOffset] = byte(f.PayloadLength >> 56)
			currFrame[3 + payloadOffset] = byte(f.PayloadLength >> 48)
			currFrame[4 + payloadOffset] = byte(f.PayloadLength >> 40)
			currFrame[5 + payloadOffset] = byte(f.PayloadLength >> 32)
			currFrame[6 + payloadOffset] = byte(f.PayloadLength >> 24)
			currFrame[7 + payloadOffset] = byte(f.PayloadLength >> 16)
			currFrame[8 + payloadOffset] = byte(f.PayloadLength >> 8)
			currFrame[9 + payloadOffset] = byte(f.PayloadLength)
		}

		totalOffset := maskOffset + payloadOffset
		for i, v := range f.Payload {
			currFrame[totalOffset + i] |= v
		}
	}
	return []byte{}
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
