package tools

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/USACE/filestore"
	"github.com/dewberry/gdal"
)

// GeoData ...
type GeoData struct {
	Features     map[string]Features
	Georeference int
}

// Features ...
type Features struct {
	Rivers              []VectorLayer
	XS                  []VectorLayer
	Banks               []VectorLayer
	StorageAreas        []VectorLayer
	TwoDAreas           []VectorLayer
	HydraulicStructures []VectorLayer
	Connections         []VectorLayer
	BCLines             []VectorLayer
	BreakLines          []VectorLayer
}

// VectorLayer ...
type VectorLayer struct {
	FeatureName string                 `json:"feature_name"`
	Fields      map[string]interface{} `json:"fields"`
	Geometry    []uint8                `json:"geometry"`
}

type xyzPoint struct {
	x float64
	y float64
	z float64
}

var unitConsistencyMap map[string]string = map[string]string{
	"English Units": "US survey foot",
	"SI Units":      "metre"}

// checkUnitConsistency checks that the unit system used by the model and its coordinate reference system are the same
func checkUnitConsistency(modelUnits string, sourceCRS string) error {
	sourceSpRef := gdal.CreateSpatialReference(sourceCRS)

	if crsUnits, ok := sourceSpRef.AttrValue("UNIT", 0); ok {
		if unitConsistencyMap[modelUnits] == crsUnits {
			return nil
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
				val1, err := strconv.ParseFloat(strings.TrimSpace(line[s:s+valueWidth]), 64)
				if err != nil {
					return pairs, err
				}
				val2, err := strconv.ParseFloat(strings.TrimSpace(line[s+valueWidth:s+stride]), 64)
				if err != nil {
					return pairs, err
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
				return pairs, err
			}
			pairs, err = dataPairsfromTextBlock(sc, nPairs, colWidth, valueWidth)
			if err != nil {
				return pairs, err
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
		return transform, err
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
		return s, err
	}
	num := reg.ReplaceAllString(s, "")
	return num, nil
}

func getRiverCenterline(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorLayer, error) {
	riverReach := strings.Split(rightofEquals(sc.Text()), ",")
	layer := VectorLayer{FeatureName: fmt.Sprintf("%s, %s", strings.TrimSpace(riverReach[0]), strings.TrimSpace(riverReach[1]))}

	xyPairs, err := getDataPairsfromTextBlock("Reach XY=", sc, 64, 16)
	if err != nil {
		return layer, err
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
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

func getXSBanks(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (VectorLayer, []VectorLayer, error) {
	bankLayers := []VectorLayer{}

	xsLayer, xyPairs, startingStation, err := getXS(sc, transform, riverReachName)
	if err != nil {
		return xsLayer, bankLayers, err
	}
	log.Println("Extracted cross-section")
	if xsLayer.Fields["CutLineProfileMatch"].(bool) {
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "Bank Sta=") {
				bankLayers, err = getBanks(line, transform, xsLayer, xyPairs, startingStation)
				if err != nil {
					return xsLayer, bankLayers, err
				}
				break
			}
		}
	}
	log.Println("Extracted banks")
	return xsLayer, bankLayers, err
}

func getXS(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (VectorLayer, [][2]float64, float64, error) {
	xyPairs := [][2]float64{}
	layer := VectorLayer{Fields: map[string]interface{}{}}
	layer.Fields["RiverReachName"] = riverReachName
	layer.Fields["CutLineProfileMatch"] = false

	compData := strings.Split(rightofEquals(sc.Text()), ",")

	xsName, err := toNumeric(compData[1])
	if err != nil {
		return layer, xyPairs, 0.0, err
	}
	layer.FeatureName = xsName

	xyPairs, err = getDataPairsfromTextBlock("XS GIS Cut Line", sc, 64, 16)
	if err != nil {
		return layer, xyPairs, 0.0, err
	}

	if len(xyPairs) < 2 {
		err = errors.New("the cross-section cutline could not be extracted, check that the geometry file contains cutlines")
		return layer, xyPairs, 0.0, err
	}

	xyzLineString := gdal.Create(gdal.GT_LineString25D)
	for _, pair := range xyPairs {
		xyzLineString.AddPoint(pair[0], pair[1], 0.0)
	}
	lenCutLine := xyzLineString.Length()

	mzPairs, err := getDataPairsfromTextBlock("#Sta/Elev", sc, 80, 8)
	if err != nil {
		return layer, xyPairs, mzPairs[0][0], err
	}

	if len(mzPairs) >= 2 {
		lenProfile := mzPairs[len(mzPairs)-1][0] - mzPairs[0][0]
		if math.Abs(lenProfile-lenCutLine) <= 0.1 {
			xyzPoints := attributeZ(xyPairs, mzPairs)
			xyzLineString = gdal.Create(gdal.GT_LineString25D)
			for _, point := range xyzPoints {
				xyzLineString.AddPoint(point.x, point.y, point.z)
			}
			layer.Fields["CutLineProfileMatch"] = true
		}
	}

	xyzLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped
	yxzLineString := flipXYLineString25D(xyzLineString)

	multiLineString := yxzLineString.ForceToMultiLineString()
	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return layer, xyPairs, mzPairs[0][0], err
	}
	layer.Geometry = wkb
	return layer, xyPairs, mzPairs[0][0], err
}

func getBanks(line string, transform gdal.CoordinateTransform, xsLayer VectorLayer, xyPairs [][2]float64, startingStation float64) ([]VectorLayer, error) {
	layers := []VectorLayer{}

	bankStations := strings.Split(rightofEquals(line), ",")
	for _, s := range bankStations {
		layer := VectorLayer{FeatureName: strings.TrimSpace(s), Fields: map[string]interface{}{}}
		layer.Fields["RiverReachName"] = xsLayer.Fields["RiverReachName"]
		layer.Fields["xsName"] = xsLayer.FeatureName
		bankStation, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return layers, err
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
			return layers, err
		}
		layer.Geometry = wkb
		layers = append(layers, layer)
	}
	return layers, nil
}

func getStorageArea(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorLayer, error) {
	layer := VectorLayer{FeatureName: strings.TrimSpace(strings.Split(rightofEquals(sc.Text()), ",")[0])}

	xyPairs, err := getDataPairsfromTextBlock("Storage Area Surface Line=", sc, 32, 16)
	if err != nil {
		return layer, err
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
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

// Extract name and geometry from BreakLine text block and return as Vector Layer
func getBreakLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorLayer, error) {
	blName := rightofEquals(sc.Text())
	layer := VectorLayer{FeatureName: blName}

	xyPairs, err := getDataPairsfromTextBlock("BreakLine Polyline=", sc, 64, 16)
	if err != nil {
		return layer, err
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return layer, errors.New("Invalid Line Geometry")
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
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

// Extract name and geometry from Boundary Condition text block and return as Vector Layer
func getBCLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorLayer, error) {
	bcName := rightofEquals(sc.Text())
	layer := VectorLayer{FeatureName: bcName}

	xyPairs, err := getDataPairsfromTextBlock("BC Line Arc=", sc, 64, 16)
	if err != nil {
		return layer, err
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return layer, errors.New("Invalid Line Geometry")
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
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

// Extract name and geometry from Connection text block and return as Vector Layer
func getConnectionLine(sc *bufio.Scanner, transform gdal.CoordinateTransform) (VectorLayer, error) {
	layer := VectorLayer{FeatureName: strings.TrimSpace(strings.Split(rightofEquals(sc.Text()), ",")[0])}

	xyPairs, err := getDataPairsfromTextBlock("Connection Line=", sc, 64, 16)
	if err != nil {
		return layer, err
	}

	// If less than 2 xyPairs, it is not a valid line.
	if len(xyPairs) < 2 {
		return layer, errors.New("Invalid Line Geometry")
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
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

// GetGeospatialData ...
func GetGeospatialData(gd *GeoData, fs filestore.FileStore, geomFilePath string, sourceCRS string, destinationCRS int) error {
	geomFileName := filepath.Base(geomFilePath)
	f := Features{}
	riverReachName := ""
	log.Println("Extracting geospatial data from:", geomFilePath)

	file, err := fs.GetObject(geomFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sc := bufio.NewScanner(file)

	transform, err := getTransform(sourceCRS, destinationCRS)
	if err != nil {
		return err
	}

	for sc.Scan() {
		line := sc.Text()

		switch {
		case strings.HasPrefix(line, "River Reach="):
			riverLayer, err := getRiverCenterline(sc, transform)
			if err != nil {
				return err
			}
			f.Rivers = append(f.Rivers, riverLayer)
			riverReachName = riverLayer.FeatureName
			log.Println("Extracted river centerline")

		case strings.HasPrefix(line, "Storage Area="):
			storageAreaLayer, err := getStorageArea(sc, transform)
			if err != nil {
				return err
			}
			f.StorageAreas = append(f.StorageAreas, storageAreaLayer)
			log.Println("Extracted storage area")

		case strings.HasPrefix(line, "Type RM Length L Ch R = 1"):
			xsLayer, bankLayers, err := getXSBanks(sc, transform, riverReachName)
			if err != nil {
				return err
			}
			f.XS = append(f.XS, xsLayer)
			f.Banks = append(f.Banks, bankLayers...)
			log.Println("Extracted banks and cross-sections")

		case strings.HasPrefix(line, "BreakLine Name="):
			blLayer, err := getBreakLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println(blLayer.FeatureName, err.Error())
				default:
					return err
				}
			default:
				f.BreakLines = append(f.BreakLines, blLayer)
				log.Println("Extracted break line")
			}

		case strings.HasPrefix(line, "BC Line Name="):
			bcLayer, err := getBCLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println(bcLayer.FeatureName, err.Error())
				default:
					return err
				}
			default:
				f.BCLines = append(f.BCLines, bcLayer)
				log.Println("Extracted boundary conditions line")
			}

		case strings.HasPrefix(line, "Connection="):
			connLayer, err := getConnectionLine(sc, transform)
			switch {
			case err != nil:
				switch {
				case err.Error() == "Invalid Line Geometry":
					log.Println(connLayer.FeatureName, err.Error())
				default:
					return err
				}
			default:
				f.Connections = append(f.Connections, connLayer)
				log.Println("Extracted connection line")
			}
		}
	}

	gd.Features[geomFileName] = f
	return nil
}
