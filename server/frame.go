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
			frameSize += 1
			maskOffset += 1
		} else if f.PayloadLength > 65536 {
			frameSize += 4
			maskOffset += 4
		}

		currFrame := make([]byte, frameSize)

		if f.FIN {
			currFrame[0] |= 0x80
		}

		currFrame[0] |= f.Opcode

		if f.Mask {
			currFrame[1] |= 0x80
		}

		if f.Mask {
			currFrame[2+maskOffset] |= f.MaskKey[0]
			currFrame[3+maskOffset] |= f.MaskKey[1]
			currFrame[4+maskOffset] |= f.MaskKey[2]
			currFrame[5+maskOffset] |= f.MaskKey[3]
		}

		// --- FIRST BYTE IS SET ---

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
	}
	return []byte{}
}

// Turns payload (byte array) into frames
func PayloadToFrames([]byte) []Frame {

	return []Frame{}
}
func PrintBits() {

}
