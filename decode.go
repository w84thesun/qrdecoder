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
