package tools

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/USACE/filestore"
	"github.com/dewberry/gdal"
	"github.com/go-errors/errors" // warning: replaces standard errors
)

// GeoData ...
type GeoData struct {
	Features     map[string]Features
	Georeference int
}

// Features ...
type Features struct {
	Rivers              []VectorFeature
	XS                  []VectorFeature
	Banks               []VectorFeature
	StorageAreas        []VectorFeature
	TwoDAreas           []VectorFeature
	HydraulicStructures []VectorFeature
	Connections         []VectorFeature
	BCLines             []VectorFeature
	BreakLines          []VectorFeature
}

// VectorFeature ...
type VectorFeature struct {
	FeatureName string                 `json:"feature_name"`
	Fields      map[string]interface{} `json:"fields"`
	Geometry    []uint8                `json:"geometry"`
}

type xyzPoint struct {
	x float64
	y float64
	z float64
}

var unitConsistencyGroups [][]string = [][]string{{"english units", "us survey foot", "foot_us", "foot us", "us foot"}, {"si units", "metre", "meter"}}

// checkUnitConsistency checks that the unit system used by the model and its coordinate reference system are the same
func checkUnitConsistency(modelUnits string, sourceCRS string) error {
	sourceSpRef := gdal.CreateSpatialReference(sourceCRS)

	if crsUnits, ok := sourceSpRef.AttrValue("UNIT", 0); ok {
		for _, unitsSet := range unitConsistencyGroups {
			if stringInSlice(strings.ToLower(modelUnits), unitsSet) && stringInSlice(strings.ToLower(crsUnits), unitsSet) {
				return nil
			}
		}
		return errors.New("The unit system of the model and coordinate reference system are inconsistent")
	}
	return errors.New("Unable to check unit consistency, could not identify the coordinate reference system's units")
}

func dataPairsfromTextBlock(sc *bufio.Scanner, nPairs int, colWidth int, valueWidth int) ([][2]float64, error) {
	var stride int = valueWidth * 2
	pairs := [][2]float64{}
out:
	for sc.Scan() {
		line := sc.Text()
		for s := 0; s < colWidth; {
			if len(line) > s {
				val1, err := parseFloat(strings.TrimSpace(line[s:s+valueWidth]), 64)
				if err != nil {
					return pairs, errors.Wrap(err, 0)
				}
				val2, err := parseFloat(strings.TrimSpace(line[s+valueWidth:s+stride]), 64)
				if err != nil {
					return pairs, errors.Wrap(err, 0)
				}
				pairs = append(pairs, [2]float64{val1, val2})
				if len(pairs) == nPairs {
					break out
				}
			} else {
				break
			}
			s += stride
		}
	}
	return pairs, nil
}

func getDataPairsfromTextBlock(nDataPairsLine string, sc *bufio.Scanner, colWidth int, valueWidth int) ([][2]float64, error) {
	pairs := [][2]float64{}
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, nDataPairsLine) {
			nPairs, err := strconv.Atoi(rightofEquals(line))
			if err != nil {
				return pairs, errors.Wrap(err, 0)
			}
			pairs, err = dataPairsfromTextBlock(sc, nPairs, colWidth, valueWidth)
			if err != nil {
				return pairs, errors.Wrap(err, 0)
			}
			break
		}
	}
	return pairs, nil
}

// distance returns the distance along a straight line in euclidean space
func distance(p0, p1 [2]float64) float64 {
	result := math.Sqrt(math.Pow((p1[0]-p0[0]), 2) + math.Pow((p1[1]-p0[1]), 2))
	return result
}

// pointAtDistance returns a new point along a straight line in euclidean space
// at a specified distance
func pointAtDistance(p0, p1 [2]float64, delta float64) [2]float64 {
	distanceRatio := delta / distance(p0, p1)
	newX := (1-distanceRatio)*p0[0] + distanceRatio*p1[0]
	newY := (1-distanceRatio)*p0[1] + distanceRatio*p1[1]
	return [2]float64{newX, newY}
}

// interpZ creates a new point a given distance along a line composed
// of many segments.
func interpXY(xyPairs [][2]float64, d float64) [2]float64 {
	// newPoint is an x, y pair
	var newPoint [2]float64
	lineSegments := len(xyPairs) - 1
	lineLength := 0.0

findLineSegment:
	for i := 0; i < lineSegments; i++ {
		p0, p1 := xyPairs[i], xyPairs[i+1]
		lineLength += distance(p0, p1)

		switch {
		case lineLength > d:
			delta := distance(p0, p1) - (lineLength - d)
			newPoint = pointAtDistance(p0, p1, delta)
			break findLineSegment

		default:
			continue
		}
	}
	if d >= lineLength {
		if d-lineLength <= 0.1 {
			return xyPairs[len(xyPairs)-1]
		}
		fmt.Printf("The interpolated point has a station of %v while the xy line is %v long", d, lineLength)
	}
	return newPoint
}

// attributeZ using station from cross-section line and gis coordinates
func attributeZ(xyPairs [][2]float64, mzPairs [][2]float64) []xyzPoint {
	points := []xyzPoint{}
	startingStation := mzPairs[0][0]

	for _, mzPair := range mzPairs {
		newPoint := interpXY(xyPairs, mzPair[0]-startingStation)
		if newPoint[0] != 0 && newPoint[1] != 0 {
			points = append(points, xyzPoint{newPoint[0], newPoint[1], mzPair[1]})
		} else {
			fmt.Printf("Interpolated point has an xy value of (%v, %v). ", newPoint[0], newPoint[1])
		}
	}
	return points
}

func getTransform(sourceCRS string, destinationCRS int) (gdal.CoordinateTransform, error) {
	transform := gdal.CoordinateTransform{}
	sourceSpRef := gdal.CreateSpatialReference(sourceCRS)

	destinationSpRef := gdal.CreateSpatialReference("")
	if err := destinationSpRef.FromEPSG(destinationCRS); err != nil {
		return transform, errors.Wrap(err, 0)
	}
	transform = gdal.CreateCoordinateTransform(sourceSpRef, destinationSpRef)
	return transform, nil
}

func flipXYLineString(xyLineString gdal.Geometry) gdal.Geometry {
	yxLineString := gdal.Create(gdal.GT_LineString)
	nPoints := xyLineString.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyLineString.Point(i)
		yxLineString.AddPoint2D(y, x)
	}
	xyLineString.Destroy()
	return yxLineString
}

func flipXYLineString25D(xyzLineString gdal.Geometry) gdal.Geometry {
	yxzLineString := gdal.Create(gdal.GT_LineString25D)
	nPoints := xyzLineString.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, z := xyzLineString.Point(i)
		yxzLineString.AddPoint(y, x, z)
	}
	xyzLineString.Destroy()
	return yxzLineString
}

func flipXYLinearRing(xyLinearRing gdal.Geometry) gdal.Geometry {
	yxLinearRing := gdal.Create(gdal.GT_LinearRing)
	nPoints := xyLinearRing.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyLinearRing.Point(i)
		yxLinearRing.AddPoint2D(y, x)
	}
	xyLinearRing.Destroy()
	return yxLinearRing
}

func flipXYPoint(xyPoint gdal.Geometry) gdal.Geometry {
	yxPoint := gdal.Create(gdal.GT_Point)
	nPoints := xyPoint.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyPoint.Point(i)
		yxPoint.AddPoint2D(y, x)
	}
	xyPoint.Destroy()
	return yxPoint
}

func toNumeric(s string) (string, error) {
	reg, err := regexp.Compile("[^.0-9]+")
	if err != nil {
		return s, errors.Wrap(err, 0)
	}
	num := reg.ReplaceAllString(s, "")
	return num, nil
}

func getRiverCenterline(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorFeature, error) {
	riverReach := strings.Split(rightofEquals(sc.Text()), ",")
	feature := VectorFeature{FeatureName: fmt.Sprintf("%s, %s", strings.TrimSpace(riverReach[0]), strings.TrimSpace(riverReach[1]))}

	xyPairs, err := getDataPairsfromTextBlock("Reach XY=", sc, 64, 16)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}

	xyLineString := gdal.Create(gdal.GT_LineString)
	for _, pair := range xyPairs {
		xyLineString.AddPoint2D(pair[0], pair[1])
	}

	xyLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLineString := flipXYLineString(xyLineString)

	multiLineString := yxLineString.ForceToMultiLineString()

	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Geometry = wkb
	return feature, nil
}

func getXSBanks(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (VectorFeature, []VectorFeature, error) {
	bankLayer := []VectorFeature{}

	xsFeature, xyPairs, startingStation, err := getXS(sc, transform, riverReachName)
	if err != nil {
		return xsFeature, bankLayer, errors.Wrap(err, 0)
	}

	if xsFeature.Fields["CutLineProfileMatch"].(bool) {
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "Bank Sta=") {
				bankLayer, err = getBanks(line, transform, xsFeature, xyPairs, startingStation)
				if err != nil {
					return xsFeature, bankLayer, errors.Wrap(err, 0)
				}
				break
			}
		}
	}

	return xsFeature, bankLayer, nil
}

func getXS(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (VectorFeature, [][2]float64, float64, error) {
	xyPairs := [][2]float64{}
	feature := VectorFeature{Fields: map[string]interface{}{}}
	feature.Fields["RiverReachName"] = riverReachName
	feature.Fields["CutLineProfileMatch"] = false

	compData := strings.Split(rightofEquals(sc.Text()), ",")

	xsName, err := toNumeric(compData[1])
	if err != nil {
		return feature, xyPairs, 0.0, errors.Wrap(err, 0)
	}
	feature.FeatureName = xsName

	xyPairs, err = getDataPairsfromTextBlock("XS GIS Cut Line", sc, 64, 16)
	if err != nil {
		return feature, xyPairs, 0.0, errors.Wrap(err, 0)
	}

	if len(xyPairs) < 2 {
		err = errors.New("the cross-section cutline could not be extracted, check that the geometry file contains cutlines")
		return feature, xyPairs, 0.0, errors.Wrap(err, 0)
	}

	xyzLineString := gdal.Create(gdal.GT_LineString25D)
	for _, pair := range xyPairs {
		xyzLineString.AddPoint(pair[0], pair[1], 0.0)
	}
	lenCutLine := xyzLineString.Length()

	mzPairs, err := getDataPairsfromTextBlock("#Sta/Elev", sc, 80, 8)
	if err != nil {
		return feature, xyPairs, mzPairs[0][0], errors.Wrap(err, 0)
	}

	if len(mzPairs) >= 2 {
		lenProfile := mzPairs[len(mzPairs)-1][0] - mzPairs[0][0]
		if math.Abs(lenProfile-lenCutLine) <= 0.1 {
			xyzPoints := attributeZ(xyPairs, mzPairs)
			xyzLineString = gdal.Create(gdal.GT_LineString25D)
			for _, point := range xyzPoints {
				xyzLineString.AddPoint(point.x, point.y, point.z)
			}
			feature.Fields["CutLineProfileMatch"] = true
		}
	}

	xyzLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped
	yxzLineString := flipXYLineString25D(xyzLineString)

	multiLineString := yxzLineString.ForceToMultiLineString()
	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return feature, xyPairs, mzPairs[0][0], errors.Wrap(err, 0)
	}
	feature.Geometry = wkb
	return feature, xyPairs, mzPairs[0][0], nil
}

func getBanks(line string, transform gdal.CoordinateTransform, xsFeature VectorFeature, xyPairs [][2]float64, startingStation float64) ([]VectorFeature, error) {
	layer := []VectorFeature{}

	bankStations := strings.Split(rightofEquals(line), ",")
	for _, s := range bankStations {
		feature := VectorFeature{FeatureName: strings.TrimSpace(s), Fields: map[string]interface{}{}}
		feature.Fields["RiverReachName"] = xsFeature.Fields["RiverReachName"]
		feature.Fields["xsName"] = xsFeature.FeatureName
		bankStation, err := parseFloat(s, 64)
		if err != nil {
			return layer, errors.Wrap(err, 0)
		}
		bankXY := interpXY(xyPairs, bankStation-startingStation)
		xyPoint := gdal.Create(gdal.GT_Point)
		xyPoint.AddPoint2D(bankXY[0], bankXY[1])
		xyPoint.Transform(transform)
		// This is a temporary fix since the x and y values need to be flipped
		yxPoint := flipXYPoint(xyPoint)
		multiPoint := yxPoint.ForceToMultiPoint()
		wkb, err := multiPoint.ToWKB()
		if err != nil {
			return layer, errors.Wrap(err, 0)
		}
		feature.Geometry = wkb
		layer = append(layer, feature)
	}
	return layer, nil
}

func getArea(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorFeature, string, error) {
	feature := VectorFeature{FeatureName: strings.TrimSpace(strings.Split(rightofEquals(sc.Text()), ",")[0])}

	xyPairs, err := getDataPairsfromTextBlock("Storage Area Surface Line=", sc, 32, 16)
	if err != nil {
		return feature, "", errors.Wrap(err, 0)
	}

	is2D, err := getAreaType(sc)
	if err != nil {
		return feature, "", errors.Wrap(err, 0)
	}

	xyLinearRing := gdal.Create(gdal.GT_LinearRing)
	for _, pair := range xyPairs {
		xyLinearRing.AddPoint2D(pair[0], pair[1])
	}

	xyLinearRing.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLinearRing := flipXYLinearRing(xyLinearRing)

	yxPolygon := gdal.Create(gdal.GT_Polygon)
	yxPolygon.AddGeometry(yxLinearRing)
	yxMultiPolygon := yxPolygon.ForceToMultiPolygon()
	wkb, err := yxMultiPolygon.ToWKB()
	if err != nil {
		return feature, is2D, errors.Wrap(err, 0)
	}
	feature.Geometry = wkb
	return feature, is2D, nil
}

func getAreaType(sc *bufio.Scanner) (string, error) {
	is2D := ""

	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Storage Area Is2D=") {
			is2D = rightofEquals(line)
			if is2D != "0" && is2D != "-1" {
				return "", errors.New("Cannot determine if area is storage area or 2D area.")
			}
			return is2D, nil
		}
	}

	return "", errors.New("Failed to parse area type.")
}

// Extract name and geometry from BreakLine text block and return as Vector Feature
func getBreakLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorFeature, error) {
	blName := rightofEquals(sc.Text())
	feature := VectorFeature{FeatureName: blName}

	xyPairs, err := getDataPairsfromTextBlock("BreakLine Polyline=", sc, 64, 16)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return feature, errors.New("Invalid Line Geometry")
	}

	xyLineString := gdal.Create(gdal.GT_LineString)
	for _, pair := range xyPairs {
		xyLineString.AddPoint2D(pair[0], pair[1])
	}

	xyLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLineString := flipXYLineString(xyLineString)

	multiLineString := yxLineString.ForceToMultiLineString()

	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Geometry = wkb
	return feature, nil
}

// Extract name and geometry from Boundary Condition text block and return as Vector Feature
func getBCLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorFeature, error) {
	bcName := rightofEquals(sc.Text())

	feature := VectorFeature{
		FeatureName: bcName,
		Fields:      map[string]interface{}{},
	}

	bcArea, err := getBCArea(sc)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Fields["Area"] = bcArea

	xyPairs, err := getDataPairsfromTextBlock("BC Line Arc=", sc, 64, 16)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return feature, errors.New("Invalid Line Geometry")
	}

	xyLineString := gdal.Create(gdal.GT_LineString)
	for _, pair := range xyPairs {
		xyLineString.AddPoint2D(pair[0], pair[1])
	}

	xyLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLineString := flipXYLineString(xyLineString)

	multiLineString := yxLineString.ForceToMultiLineString()

	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Geometry = wkb
	return feature, nil
}

// Get Storage / 2D Area of a BC Line
func getBCArea(sc *bufio.Scanner) (string, error) {
	bcArea := ""
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "BC Line Storage Area=") {
			bcArea = rightofEquals(line)
			return bcArea, nil
		}
	}
	return "", errors.New("Failed to parse BC Line Storage Area.")
}

// Extract name and geometry from Connection text block and return as Vector Feature
func getConnectionLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorFeature, error) {
	feature := VectorFeature{
		FeatureName: strings.TrimSpace(strings.Split(rightofEquals(sc.Text()), ",")[0]),
		Fields:      map[string]interface{}{},
	}

	xyPairs, err := getDataPairsfromTextBlock("Connection Line=", sc, 64, 16)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return feature, errors.New("Invalid Line Geometry")
	}

	xyLineString := gdal.Create(gdal.GT_LineString)
	for _, pair := range xyPairs {
		xyLineString.AddPoint2D(pair[0], pair[1])
	}

	xyLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLineString := flipXYLineString(xyLineString)

	multiLineString := yxLineString.ForceToMultiLineString()

	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Geometry = wkb

	connUpArea, connDnArea, err := getConnArea(sc)
	if err != nil {
		return feature, errors.Wrap(err, 0)
	}
	feature.Fields["Up Area"] = connUpArea
	feature.Fields["Dn Area"] = connDnArea

	return feature, nil
}

// Extract Upstream and DownStream Areas of a Connection
func getConnArea(sc *bufio.Scanner) (string, string, error) {
	connUpArea := ""
	connDnArea := ""

	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Connection Up SA=") {
			connUpArea = rightofEquals(line)
		} else if strings.HasPrefix(line, "Connection Dn SA=") {
			connDnArea = rightofEquals(line)
		}
		if connUpArea != "" && connDnArea != "" {
			return connUpArea, connDnArea, nil
		}
	}
	return "", "", errors.New("Failed to parse Connection Up/Dn Areas.")
}

// GetGeospatialData ...
func GetGeospatialData(gd *GeoData, fs filestore.FileStore, geomFilePath string, sourceCRS string, destinationCRS int) error {
	geomFileName := filepath.Base(geomFilePath)
	f := Features{}
	riverReachName := ""

	file, err := fs.GetObject(geomFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	sc := bufio.NewScanner(file)

	transform, err := getTransform(sourceCRS, destinationCRS)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for sc.Scan() {
		line := sc.Text()

		switch {
		case strings.HasPrefix(line, "River Reach="):
			riverFeature, err := getRiverCenterline(sc, transform)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			f.Rivers = append(f.Rivers, riverFeature)
			riverReachName = riverFeature.FeatureName

		case strings.HasPrefix(line, "Storage Area="):
			storageAreaFeature, aType, err := getArea(sc, transform)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			if aType == "0" {
				f.StorageAreas = append(f.StorageAreas, storageAreaFeature)
			} else if aType == "-1" {
				f.TwoDAreas = append(f.TwoDAreas, storageAreaFeature)
			}
		case strings.HasPrefix(line, "Type RM Length L Ch R = 1"):
			xsFeature, bankLayer, err := getXSBanks(sc, transform, riverReachName)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			f.XS = append(f.XS, xsFeature)
			f.Banks = append(f.Banks, bankLayer...)

		case strings.HasPrefix(line, "BreakLine Name="):
			blFeature, err := getBreakLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println("Skipped", blFeature.FeatureName, err.Error(), "Geom File:", filepath.Ext(geomFilePath))
				default:
					return errors.Wrap(err, 0)
				}
			default:
				f.BreakLines = append(f.BreakLines, blFeature)
			}

		case strings.HasPrefix(line, "BC Line Name="):
			bcFeature, err := getBCLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println("Skipped", bcFeature.FeatureName, err.Error(), "Geom File:", filepath.Ext(geomFilePath))
				default:
					return errors.Wrap(err, 0)
				}
			default:
				f.BCLines = append(f.BCLines, bcFeature)
			}

		case strings.HasPrefix(line, "Connection="):
			connFeature, err := getConnectionLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println("Skipped", connFeature.FeatureName, err.Error(), "Geom File:", filepath.Ext(geomFilePath))
				default:
					return errors.Wrap(err, 0)
				}
			default:
				f.Connections = append(f.Connections, connFeature)
			}

		}
	}

	gd.Features[geomFileName] = f
	return nil
}
