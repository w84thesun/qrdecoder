package qrcode

import (
	"image"
	"io"
	"os"
)

// QR code recognition function
func Decode(fi io.Reader) (*Matrix, error) {
	img, _, err := image.Decode(fi)
	if err != nil {
		return nil, err
	}

	qrMatrix, err := DecodeImg(img, os.TempDir())
	if err != nil {
		return nil, err
	}

	info, err := qrMatrix.FormatInfo()
	if err != nil {
		return nil, err
	}

	maskFunc := MaskFunc(info.Mask)
	unmaskMatrix := new(Matrix)

	for y, line := range qrMatrix.Points {
		var l []bool
		for x, value := range line {
			l = append(l, maskFunc(x, y) != value)
		}
		unmaskMatrix.Points = append(unmaskMatrix.Points, l)
	}

	dataArea := unmaskMatrix.DataArea()

	dataCode, err := ParseBlock(qrMatrix, GetData(unmaskMatrix, dataArea))
	if err != nil {
		return nil, err
	}

	bt, err := Bits2Bytes(dataCode, unmaskMatrix.Version())
	if err != nil {
		return nil, err
	}

	qrMatrix.Content = string(bt)

	return qrMatrix, nil
}

func ParseBlock(m *Matrix, data []bool) ([]bool, error) {
	version := m.Version()
	info, err := m.FormatInfo()
	if err != nil {
		return nil, err
	}
	var qrCodeVersion = QRcodeVersion{}
	for _, qrCV := range Versions {
		if qrCV.Level == RecoveryLevel(info.ErrorCorrectionLevel) && qrCV.Version == version {
			qrCodeVersion = qrCV
		}
	}

	var dataBlocks [][]bool
	for _, block := range qrCodeVersion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			dataBlocks = append(dataBlocks, []bool{})
		}
	}
	for {
		leftLength := len(data)
		no := 0
		for _, block := range qrCodeVersion.Block {
			for i := 0; i < block.NumBlocks; i++ {
				if len(dataBlocks[no]) < block.NumDataCodewords*8 {
					dataBlocks[no] = append(dataBlocks[no], data[0:8]...)
					data = data[8:]
				}
				no += 1
			}
		}
		if leftLength == len(data) {
			break
		}
	}

	var errorBlocks [][]bool
	for _, block := range qrCodeVersion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			errorBlocks = append(errorBlocks, []bool{})
		}
	}
	for {
		leftLength := len(data)
		no := 0
		for _, block := range qrCodeVersion.Block {
			for i := 0; i < block.NumBlocks; i++ {
				if len(errorBlocks[no]) < (block.NumCodewords-block.NumDataCodewords)*8 {
					errorBlocks[no] = append(errorBlocks[no], data[:8]...)
					if len(data) > 8 {
						data = data[8:]
					}
				}
				no += 1
			}
		}
		if leftLength == len(data) {
			break
		}
	}

	var result []byte
	for i := range dataBlocks {
		blockByte, err := QRReconstruct(Bool2Byte(dataBlocks[i]), Bool2Byte(errorBlocks[i]))
		if err != nil {
			return nil, err
		}
		result = append(result, blockByte[:len(Bool2Byte(dataBlocks[i]))]...)
	}
	return Byte2Bool(result), nil
}
