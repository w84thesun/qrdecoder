package qrcode

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"

	"github.com/maruel/rs"
)

type PositionDetectionPatterns struct {
	TopLeft *PointGroup
	Right   *PointGroup
	Bottom  *PointGroup
}

type PointGroup struct {
	Group    []Point
	GroupMap map[Point]bool
	Min      Point
	Max      Point
	Center   Point
	IsHollow bool
}

type PointsMatrix [][]bool

func (p PointsMatrix) Copy() PointsMatrix {
	newP := make(PointsMatrix, len(p))

	for i, line := range p {
		newP[i] = make([]bool, len(line))
		copy(newP[i], line)
	}

	return newP
}

type Matrix struct {
	OrgImage  image.Image
	OrgSize   image.Rectangle
	OrgPoints PointsMatrix
	Points    PointsMatrix
	Size      image.Rectangle
	Data      []bool
	Content   string
}

func (mx *Matrix) AtOrgPoints(x, y int) bool {
	if y >= 0 && y < len(mx.OrgPoints) {
		if x >= 0 && x < len(mx.OrgPoints[y]) {
			return mx.OrgPoints[y][x]
		}
	}
	return false
}

type FormatInfo struct {
	ErrorCorrectionLevel, Mask int
}

func (mx *Matrix) FormatInfo() (*FormatInfo, error) {
	fi1 := []Point{
		{0, 8}, {1, 8}, {2, 8}, {3, 8},
		{4, 8}, {5, 8}, {7, 8},
		{8, 8}, {8, 7}, {8, 5}, {8, 4},
		{8, 3}, {8, 2}, {8, 1}, {8, 0},
	}
	maskedFileData := mx.GetBin(fi1)
	unmaskFileData := maskedFileData ^ 0x5412
	if bch(unmaskFileData) == 0 {
		return &FormatInfo{
			ErrorCorrectionLevel: unmaskFileData >> 13,
			Mask:                 unmaskFileData >> 10 & 7,
		}, nil
	}
	length := len(mx.Points)
	fi2 := []Point{
		{8, length - 1}, {8, length - 2}, {8, length - 3}, {8, length - 4},
		{8, length - 5}, {8, length - 6}, {8, length - 7},
		{length - 8, 8}, {length - 7, 8}, {length - 6, 8}, {length - 5, 8},
		{length - 4, 8}, {length - 3, 8}, {length - 2, 8}, {length - 1, 8},
	}
	maskedFileData = mx.GetBin(fi2)
	unmaskFileData = maskedFileData ^ 0x5412
	if bch(unmaskFileData) == 0 {
		return &FormatInfo{
			ErrorCorrectionLevel: unmaskFileData >> 13,
			Mask:                 unmaskFileData >> 10 & 7,
		}, nil
	}
	return nil, errors.New("not found error correction level and mask")
}

func (mx *Matrix) AtPoints(x, y int) bool {
	if y >= 0 && y < len(mx.Points) {
		if x >= 0 && x < len(mx.Points[y]) {
			return mx.Points[y][x]
		}
	}
	return false
}

func (mx *Matrix) GetBin(poss []Point) int {
	var fileData int
	for _, pos := range poss {
		if mx.AtPoints(pos.X, pos.Y) {
			fileData = fileData<<1 + 1
		} else {
			fileData = fileData << 1
		}
	}
	return fileData
}

func (mx *Matrix) Version() int {
	width := len(mx.Points)
	return (width-21)/4 + 1
}

type Point struct {
	X int
	Y int
}

func bch(org int) int {
	var g = 0x537
	for i := 4; i > -1; i-- {
		if org&(1<<(uint(i+10))) > 0 {
			org ^= g << uint(i)
		}
	}
	return org
}

func (mx *Matrix) DataArea() *Matrix {
	da := new(Matrix)
	width := len(mx.Points)
	maxPos := width - 1
	for _, line := range mx.Points {
		var l []bool
		for range line {
			l = append(l, true)
		}
		da.Points = append(da.Points, l)
	}
	// Position Detection Pattern is a pattern used to mark the size of the QR code rectangle.
	// These three position detection patterns have white borders called Separators for Position Detection Patterns. The reason for three instead of four is that three can mark a rectangle.
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if y < len(mx.Points) && x < len(mx.Points[y]) {
				da.Points[y][x] = false // Top left
			}
		}
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 8; x++ {
			if y < len(mx.Points) && maxPos-x < len(mx.Points[y]) {
				da.Points[y][maxPos-x] = false // Top right
			}
		}
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 9; x++ {
			if maxPos-y < len(mx.Points) && x < len(mx.Points[y]) {
				da.Points[maxPos-y][x] = false // Bottom left
			}
		}
	}
	// Timing Patterns are also used for positioning. The reason is that there are 40 sizes of QR codes, and when the size is too large, a standard line is needed, otherwise it may be scanned crookedly.
	for i := 0; i < width; i++ {
		if 6 < len(mx.Points) && i < len(mx.Points[6]) {
			da.Points[6][i] = false
		}
		if i < len(mx.Points) && 6 < len(mx.Points[i]) {
			da.Points[i][6] = false
		}
	}
	// Alignment Patterns are needed for QR codes of Version 2 and above (including Version 2), also for positioning.
	version := da.Version()
	Alignments := AlignmentPatternCenter[version]
	for _, AlignmentX := range Alignments {
		for _, AlignmentY := range Alignments {
			if (AlignmentX == 6 && AlignmentY == 6) || (maxPos-AlignmentX == 6 && AlignmentY == 6) || (AlignmentX == 6 && maxPos-AlignmentY == 6) {
				continue
			}
			for y := AlignmentY - 2; y <= AlignmentY+2; y++ {
				for x := AlignmentX - 2; x <= AlignmentX+2; x++ {
					if y < len(mx.Points) && x < len(mx.Points[y]) {
						da.Points[y][x] = false
					}
				}
			}
		}
	}
	// Version Information is needed for versions >= 7, reserving two 3 x 6 areas to store some version information.
	if version >= 7 {
		for i := maxPos - 10; i < maxPos-7; i++ {
			for j := 0; j < 6; j++ {
				if i < len(mx.Points) && j < len(mx.Points[i]) {
					da.Points[i][j] = false
				}
				if j < len(mx.Points) && i < len(mx.Points[j]) {
					da.Points[j][i] = false
				}
			}
		}
	}
	return da
}

func NewPositionDetectionPattern(PDPs [][]*PointGroup) (*PositionDetectionPatterns, error) {
	if len(PDPs) < 3 {
		return nil, errors.New("lost Position Detection Pattern")
	}
	var pdpGroups []*PointGroup
	for _, pdp := range PDPs {
		pdpGroups = append(pdpGroups, PossListToGroup(pdp))
	}
	var ks []*K
	for i, firstPDPGroup := range pdpGroups {
		for j, lastPDPGroup := range pdpGroups {
			if i == j {
				continue
			}
			k := &K{FirstPosGroup: firstPDPGroup, LastPosGroup: lastPDPGroup}
			Radian(k)
			ks = append(ks, k)
		}
	}
	var Offset float64 = 360
	var KF, KL *K
	for i, kf := range ks {
		for j, kl := range ks {
			if i == j {
				continue
			}
			if kf.FirstPosGroup != kl.FirstPosGroup {
				continue
			}
			offset := IsVertical(kf, kl)
			if offset < Offset {
				Offset = offset
				KF = kf
				KL = kl
			}
		}
	}
	positionDetectionPatterns := new(PositionDetectionPatterns)
	positionDetectionPatterns.TopLeft = KF.FirstPosGroup
	positionDetectionPatterns.Bottom = KL.LastPosGroup
	positionDetectionPatterns.Right = KF.LastPosGroup
	return positionDetectionPatterns, nil
}

func PossListToGroup(groups []*PointGroup) *PointGroup {
	var newGroup []Point
	for _, group := range groups {
		newGroup = append(newGroup, group.Group...)
	}
	return NewPointGroup(newGroup)
}

type K struct {
	FirstPosGroup *PointGroup
	LastPosGroup  *PointGroup
	K             float64
}

func Radian(k *K) {
	x, y := k.LastPosGroup.Center.X-k.FirstPosGroup.Center.X, k.LastPosGroup.Center.Y-k.FirstPosGroup.Center.Y
	k.K = math.Atan2(float64(y), float64(x))
}

func IsVertical(kf, kl *K) (offset float64) {
	dk := kl.K - kf.K
	offset = math.Abs(dk - math.Pi/2)
	return
}

func NewPointGroup(group []Point) *PointGroup {
	pm := make(map[Point]bool)

	for _, point := range group {
		pm[point] = true
	}

	min, max := Rectangle(group)

	return &PointGroup{
		Group:    group,
		Center:   CenterPoint(group),
		GroupMap: pm,
		Min:      min,
		Max:      max,
		IsHollow: Hollow(pm, min, max),
	}
}

func Rectangle(group []Point) (Point, Point) {
	minX, maxX, minY, maxY := group[0].X, group[0].X, group[0].Y, group[0].Y

	for _, pos := range group {
		if pos.X < minX {
			minX = pos.X
		}
		if pos.X > maxX {
			maxX = pos.X
		}
		if pos.Y < minY {
			minY = pos.Y
		}
		if pos.Y > maxY {
			maxY = pos.Y
		}
	}

	pMin := Point{X: minX, Y: minY}
	pMax := Point{X: maxX, Y: maxY}

	return pMin, pMax
}

func CenterPoint(group []Point) Point {
	sumX, sumY := 0, 0
	for _, pos := range group {
		sumX += pos.X
		sumY += pos.Y
	}
	meanX := sumX / len(group)
	meanY := sumY / len(group)
	return Point{X: meanX, Y: meanY}
}

func MaskFunc(code int) func(x, y int) bool {
	switch code {
	case 0: // 000
		return func(x, y int) bool {
			return (x+y)%2 == 0
		}
	case 1: // 001
		return func(x, y int) bool {
			return y%2 == 0
		}
	case 2: // 010
		return func(x, y int) bool {
			return x%3 == 0
		}
	case 3: // 011
		return func(x, y int) bool {
			return (x+y)%3 == 0
		}
	case 4: // 100
		return func(x, y int) bool {
			return (y/2+x/3)%2 == 0
		}
	case 5: // 101
		return func(x, y int) bool {
			return (x*y)%2+(x*y)%3 == 0
		}
	case 6: // 110
		return func(x, y int) bool {
			return ((x*y)%2+(x*y)%3)%2 == 0
		}
	case 7: // 111
		return func(x, y int) bool {
			return ((x+y)%2+(x*y)%3)%2 == 0
		}
	}
	return func(x, y int) bool {
		return false
	}
}

func SplitGroup(pointMatrix *PointsMatrix, centerX, centerY int, around *[]Point) {
	maxY := len(*pointMatrix) - 1
	for y := -1; y < 2; y++ {
		for x := -1; x < 2; x++ {
			hereY := centerY + y
			if hereY < 0 || hereY > maxY {
				continue
			}

			hereX := centerX + x
			maxX := len((*pointMatrix)[hereY]) - 1

			if hereX < 0 || hereX > maxX {
				continue
			}

			v := (*pointMatrix)[hereY][hereX]
			if v {
				(*pointMatrix)[hereY][hereX] = false

				*around = append(*around, Point{hereX, hereY})
			}
		}
	}
}

func Hollow(pm map[Point]bool, minP, maxP Point) bool {
	count := len(pm)

	for y := minP.Y; y <= maxP.Y; y++ {
		min, max := -1, -1

		for x := minP.X; x <= maxP.X; x++ {
			if pm[Point{x, y}] {
				if min < 0 {
					min = x
				}

				max = x
			}
		}

		count -= (max - min + 1)
	}

	return count != 0
}

func LineWidth(positionDetectionPatterns [][]*PointGroup) float64 {
	sumWidth := 0
	for _, positionDetectionPattern := range positionDetectionPatterns {
		for _, group := range positionDetectionPattern {
			sumWidth += group.Max.X - group.Min.X + 1
			sumWidth += group.Max.Y - group.Min.Y + 1
		}
	}
	return float64(sumWidth) / 60
}

func IsPositionDetectionPattern(solidGroup, hollowGroup *PointGroup) bool {
	solidMinX, solidMaxX, solidMinY, solidMaxY := solidGroup.Min.X, solidGroup.Max.X, solidGroup.Min.Y, solidGroup.Max.Y
	minX, maxX, minY, maxY := hollowGroup.Min.X, hollowGroup.Max.X, hollowGroup.Min.Y, hollowGroup.Max.Y

	if !(solidMinX > minX && solidMaxX > minX &&
		solidMinX < maxX && solidMaxX < maxX &&
		solidMinY > minY && solidMaxY > minY &&
		solidMinY < maxY && solidMaxY < maxY) {
		return false
	}

	hollowCenter := hollowGroup.Center

	return hollowCenter.X > solidMinX && hollowCenter.X < solidMaxX &&
		hollowCenter.Y > solidMinY && hollowCenter.Y < solidMaxY
}

func GetData(unmaskMatrix, dataArea *Matrix) []bool {
	width := len(unmaskMatrix.Points)
	var data []bool
	maxPos := width - 1
	for t := maxPos; t > 0; {
		for y := maxPos; y >= 0; y-- {
			for x := t; x >= t-1; x-- {
				if dataArea.AtPoints(x, y) {
					data = append(data, unmaskMatrix.AtPoints(x, y))
				}
			}
		}
		t = t - 2
		if t == 6 {
			t = t - 1
		}
		for y := 0; y <= maxPos; y++ {
			for x := t; x >= t-1 && x >= 0; x-- {
				if x < len(unmaskMatrix.Points[y]) && dataArea.AtPoints(x, y) {
					data = append(data, unmaskMatrix.AtPoints(x, y))
				}
			}
		}
		t = t - 2
	}
	return data
}

func Line(start, end *Point, matrix *Matrix) (line []bool) {
	if math.Abs(float64(start.X-end.X)) > math.Abs(float64(start.Y-end.Y)) {
		length := end.X - start.X
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.Y-start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k*float64(i))
				line = append(line, matrix.AtOrgPoints(x, y))
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.Y-start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k*float64(i))
				line = append(line, matrix.AtOrgPoints(x, y))
			}
		}
	} else {
		length := end.Y - start.Y
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.X-start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k*float64(i))
				line = append(line, matrix.AtOrgPoints(x, y))
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.X-start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k*float64(i))
				line = append(line, matrix.AtOrgPoints(x, y))
			}
		}
	}
	return
}

// 标线
func (mx *Matrix) CenterList(line []bool, offset int) (li []int) {
	subMap := map[int]int{}
	value := line[0]
	subLength := 0
	for _, b := range line {
		if b == value {
			subLength += 1
		} else {
			_, ok := subMap[subLength]
			if ok {
				subMap[subLength] += 1
			} else {
				subMap[subLength] = 1
			}
			value = b
			subLength = 1
		}
	}
	var maxCountSubLength float64
	var meanSubLength float64
	for k, v := range subMap {
		if float64(v) > maxCountSubLength {
			maxCountSubLength = float64(v)
			meanSubLength = float64(k)
		}
	}
	value = !line[0]
	for index, b := range line {
		if b != value {
			li = append(li, index+offset+int(meanSubLength/2))
			value = b
		}
	}
	return li

	// TODO: Multi-angle recognition
}

func ExportGroups(size image.Rectangle, hollow []*PointGroup, filename string) error {
	result := image.NewGray(size)
	for _, group := range hollow {
		for _, pos := range group.Group {
			result.Set(pos.X, pos.Y, color.White)
		}
	}
	outImg, err := os.Create(filename + ".png")
	if err != nil {
		return err
	}
	defer outImg.Close()
	return png.Encode(outImg, result)
}

func (mx *Matrix) Binarization() uint8 {
	return 128
}

func (mx *Matrix) SplitGroups() [][]Point {
	m := mx.OrgPoints.Copy()

	var groups [][]Point

	for y, line := range m {
		for x, v := range line {
			if !v {
				continue
			}

			newGroup := []Point{{x, y}}

			m[y][x] = false

			for i := range newGroup {
				v := newGroup[i]
				SplitGroup(&m, v.X, v.Y, &newGroup)
			}
			groups = append(groups, newGroup)
		}
	}

	return groups
}

func (mx *Matrix) ReadImage() {
	width := mx.OrgSize.Dx()
	height := mx.OrgSize.Dy()

	pic := image.NewGray(mx.OrgSize)

	draw.Draw(pic, mx.OrgSize, mx.OrgImage, mx.OrgImage.Bounds().Min, draw.Src)

	fz := mx.Binarization()

	for y := range height {
		var line []bool
		for x := range width {
			if pic.Pix[y*width+x] < fz {
				line = append(line, true)

				continue
			}

			line = append(line, false)
		}

		mx.OrgPoints = append(mx.OrgPoints, line)
	}
}

func QRReconstruct(data, ecc []byte) ([]byte, error) {
	_, err := rs.NewDecoder(rs.QRCodeField256).Decode(data, ecc)
	if err != nil {
		return nil, err
	}
	return data, nil
}
