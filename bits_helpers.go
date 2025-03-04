package qrcode

import "fmt"

func Byte2Bool(bl []byte) []bool {
	var result []bool
	for _, b := range bl {
		temp := make([]bool, 8)
		for i := 0; i < 8; i++ {
			if (b>>uint(i))&1 == 1 {
				temp[7-i] = true
			} else {
				temp[7-i] = false
			}

		}
		result = append(result, temp...)
	}
	return result
}

func Bits2Bytes(dataCode []bool, version int) ([]byte, error) {
	// The first 4 bits are the encoding format, the next four bits are the actual data
	mode := Bit2Int(dataCode[0:4])
	encoder, err := GetDataEncoder(version)
	if err != nil {
		return nil, err
	}
	err = encoder.SetCharModeCharDecoder(mode)
	if err != nil {
		return nil, err
	}

	modeCharDecoder := encoder.ModeCharDecoder

	return modeCharDecoder.Decode(dataCode[4:])
}

func StringBool(dataCode []bool) string {
	return StringByte(Bool2Byte(dataCode))
}

func StringByte(b []byte) string {
	var bitString string
	for i := 0; i < len(b)*8; i++ {
		if (i % 8) == 0 {
			bitString += " "
		}

		if (b[i/8] & (0x80 >> byte(i%8))) != 0 {
			bitString += "1"
		} else {
			bitString += "0"
		}
	}

	return fmt.Sprintf("numBits=%d, bits=%s", len(b)*8, bitString)
}

func Bool2Byte(dataCode []bool) []byte {
	var result []byte
	for i := 0; i < len(dataCode); {
		result = append(result, Bit2Byte(dataCode[i:i+8]))
		i += 8
	}
	return result
}
func Bit2Int(bits []bool) int {
	g := 0
	for _, i := range bits {
		g = g << 1
		if i {
			g += 1
		}
	}
	return g
}

func Bit2Byte(bits []bool) byte {
	var g uint8
	for _, i := range bits {
		g = g << 1
		if i {
			g += 1
		}
	}
	return byte(g)
}
