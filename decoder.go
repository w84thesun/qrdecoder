package qrcode

import (
	"image"
	"path/filepath"
	"strconv"
)

func DecodeImg(img image.Image, path string) (*Matrix, error) {
	matrix := &Matrix{
		OrgImage: img,
		OrgSize:  img.Bounds(),
	}

	matrix.ReadImage()

	groups := matrix.SplitGroups()
	// Determine hollow
	var hollow []*PointGroup
	// Determine solid
	var solid []*PointGroup

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		newGroup := NewPointGroup(group)

		if newGroup.IsHollow {
			hollow = append(hollow, newGroup)
		} else {
			solid = append(solid, newGroup)
		}
	}

	var positionDetectionPatterns [][]*PointGroup
	for _, solidGroup := range solid {
		for _, hollowGroup := range hollow {
			if IsPositionDetectionPattern(solidGroup, hollowGroup) {
				positionDetectionPatterns = append(positionDetectionPatterns, []*PointGroup{solidGroup, hollowGroup})
			}
		}
	}

	for i, pattern := range positionDetectionPatterns {
		ExportGroups(matrix.OrgSize, pattern, filepath.Join(path, "positionDetectionPattern"+strconv.Itoa(i)))
	}

	lineWidth := LineWidth(positionDetectionPatterns)

	pdp, err := NewPositionDetectionPattern(positionDetectionPatterns)
	if err != nil {
		return nil, err
	}

	// Top marking line
	topStart := &Point{X: pdp.TopLeft.Center.X + (int(3.5*lineWidth) + 1), Y: pdp.TopLeft.Center.Y + int(3*lineWidth)}
	topEnd := &Point{X: pdp.Right.Center.X - (int(3.5*lineWidth) + 1), Y: pdp.Right.Center.Y + int(3*lineWidth)}

	topTimePattens := Line(topStart, topEnd, matrix)

	topCL := matrix.CenterList(topTimePattens, topStart.X)

	// Left marking line
	leftStart := &Point{X: pdp.TopLeft.Center.X + int(3*lineWidth), Y: pdp.TopLeft.Center.Y + (int(3.5*lineWidth) + 1)}
	leftEnd := &Point{X: pdp.Bottom.Center.X + int(3*lineWidth), Y: pdp.Bottom.Center.Y - (int(3.5*lineWidth) + 1)}

	leftTimePattens := Line(leftStart, leftEnd, matrix)

	leftCL := matrix.CenterList(leftTimePattens, leftStart.Y)

	var qrTopCL []int
	for i := -3; i <= 3; i++ {
		qrTopCL = append(qrTopCL, pdp.TopLeft.Center.X+int(float64(i)*lineWidth))
	}

	qrTopCL = append(qrTopCL, topCL...)
	for i := -3; i <= 3; i++ {
		qrTopCL = append(qrTopCL, pdp.Right.Center.X+int(float64(i)*lineWidth))
	}

	var qrLeftCL []int
	for i := -3; i <= 3; i++ {
		qrLeftCL = append(qrLeftCL, pdp.TopLeft.Center.Y+int(float64(i)*lineWidth))
	}

	qrLeftCL = append(qrLeftCL, leftCL...)
	for i := -3; i <= 3; i++ {
		qrLeftCL = append(qrLeftCL, pdp.Bottom.Center.Y+int(float64(i)*lineWidth))
	}

	for _, y := range qrLeftCL {
		var line []bool
		for _, x := range qrTopCL {
			line = append(line, matrix.AtOrgPoints(x, y))
		}
		matrix.Points = append(matrix.Points, line)
	}

	matrix.Size = image.Rect(0, 0, len(matrix.Points), len(matrix.Points))

	return matrix, nil
}
