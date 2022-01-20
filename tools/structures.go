package tools

import (
	"bufio"
	"math"
	"strconv"
	"strings"

	"github.com/go-errors/errors" // warning: replaces standard errors
)

// Map of HEC RAS Shape index to Culvert Cross Sections
var conduitShapes map[int]string = map[int]string{
	1: "Circular",
	2: "Box",
	3: "Pipe Arch",
	4: "Ellipse",
	5: "Arch",
	6: "Semi-Circle",
	7: "Low Arch",
	8: "High Arch",
	9: "Conspan Arch"}

// Store HEC-RAS 1D Hydraulic Structures
type hydraulicStructures struct {
	River       string      `json:"River Name"`
	Reach       string      `json:"Reach Name"`
	NumXS       int         `json:"Num CrossSections"`
	CulvertData culvertData `json:"Culvert Data"`
	BridgeData  bridgeData  `json:"Bridge Data"`
	WeirData    weirData    `json:"Inline Weir Data"`
}

type culvertData struct {
	NumCulverts int        `json:"Num Culverts"`
	Culverts    []culverts `json:"Culverts"`
}

// Store HEC-RAS 1D Culverts. This is different than culverts
// associated with inline structures or connections
type culverts struct {
	Name          string
	Station       float64
	Description   string
	DeckWidth     float64     `json:"Deck Width"`
	UpHighChord   maxMinPairs `json:"Upstream High Chord"`
	UpLowChord    maxMinPairs `json:"Upstream Low Chord"`
	DownHighChord maxMinPairs `json:"Downstream High Chord"`
	DownLowChord  maxMinPairs `json:"Downstream Low Chord"`
	NumConduits   int         `json:"Num Culvert Conduits"`
	Conduits      []conduits  `json:"Culvert Conduits"`
}

// Store Min and Maximum Elevation from HEC-RAS Deck/Weir Station Elevation Block
type maxMinPairs struct {
	Max float64
	Min float64
}

// Store Culvert Groups in HEC-RAS 1D Culverts, 1D Inline Structures, and Connections
type conduits struct {
	Name       string
	NumBarrels int `json:"Num Barrels"`
	Shape      string
	Rise       float64
	Span       float64
	Length     float64
	ManningsN  float64 `json:"Mannings N"`
}

type bridgeData struct {
	NumBridges int       `json:"Num Bridges"`
	Bridges    []bridges `json:"Bridges"`
}

// Store HEC-RAS 1D Bridges
type bridges struct {
	Name          string
	Station       float64
	Description   string
	DeckWidth     float64     `json:"Deck Width"`
	UpHighChord   maxMinPairs `json:"Upstream High Chord"`
	UpLowChord    maxMinPairs `json:"Upstream Low Chord"`
	DownHighChord maxMinPairs `json:"Downstream High Chord"`
	DownLowChord  maxMinPairs `json:"Downstream Low Chord"`
	NumPiers      int         `json:"Num Piers"`
}

type weirData struct {
	NumWeirs int     `json:"Num Inline Weirs"`
	Weirs    []weirs `json:"Inline Weirs"`
}

// Store HEC-RAS 1D Inline Structures
type weirs struct {
	Name        string
	Station     float64
	Description string
	WeirWidth   float64     `json:"Weir Width"`
	WeirElev    maxMinPairs `json:"Weir Elevations"`
	NumGates    int         `json:"Num Gates"`
	Gates       []gates
	NumConduits int        `json:"Num Culvert Conduits"`
	Conduits    []conduits `json:"Culvert Conduits"`
}

// Store Gates Groups in HEC-RAS 1D Inline Structures and Connections
type gates struct {
	Name        string
	Width       float64
	Height      float64
	NumOpenings int `json:"Num Openings"`
}

// Return elevation data from HEC-RAS Station-Elevation (SE) block of text
func datafromTextBlock(hsSc *bufio.Scanner, i int, nLines int, nSkipLines int, colWidth int, valueWidth int, interval int) ([]float64, int, error) {
	values := []float64{}
	nSkipped := 0
	nProcessed := 0
	nvalues := 0
out:
	for hsSc.Scan() {
		i++
		if nSkipped < nSkipLines {
			nSkipped++
			continue
		}
		nProcessed++
		line := hsSc.Text()
		for s := 0; s < colWidth; {
			if len(line) > s {
				sVal := strings.TrimSpace(line[s : s+valueWidth])
				if sVal != "" {
					nvalues++
					if nvalues%interval == 0 {
						val, err := parseFloat(sVal, 64)
						if err != nil {
							return values, i, errors.Wrap(err, 0) 
						}
						values = append(values, val)
					}
				}
				s += valueWidth
			} else {
				if nLines == nProcessed {
					break out
				}
				break
			}

		}
		if nLines == nProcessed {
			break out
		}
	}
	return values, i, nil
}

// Return maximum and minimum elevation givin an scanner object
// with curser at definition line of SE block
func getMaxMinElev(hsSc *bufio.Scanner, i int, nLines int, nSkipLines int, colWidth int, valueWidth int, interval int) (maxMinPairs, int, error) {
	pair := maxMinPairs{}

	elevations, i, err := datafromTextBlock(hsSc, i, nLines, nSkipLines, colWidth, valueWidth, interval)

	if err != nil {
		return pair, i, errors.Wrap(err, 0) 
	}

	if len(elevations) == 0 {
		return pair, i, nil
	}

	maxElev, err := maxValue(elevations)
	if err != nil {
		return pair, i, errors.Wrap(err, 0) 
	}

	minElev, err := minValue(elevations)
	if err != nil {
		return pair, i, errors.Wrap(err, 0) 
	}

	pair = maxMinPairs{Max: maxElev, Min: minElev}
	return pair, i, nil
}

func numberofLines(nValues int, colWidth int, valueWidth int) int {
	nLines := math.Ceil(float64(nValues) / (float64(colWidth) / float64(valueWidth)))
	return int(nLines)
}

func getHighLowChord(hsSc *bufio.Scanner, i int, nElevText string, colWidth int, valueWidth int) ([2]maxMinPairs, int, error) {
	highLowPairs := [2]maxMinPairs{}

	nElev, err := strconv.Atoi(strings.TrimSpace(nElevText))
	if err != nil {
		return highLowPairs, i, errors.Wrap(err, 0) 
	}

	nLines := numberofLines(nElev, 80, 8)

	highPair, i, err := getMaxMinElev(hsSc, i, nLines, nLines, 80, 8, 1)
	if err != nil {
		return highLowPairs, i, errors.Wrap(err, 0) 
	}
	highLowPairs[0] = highPair

	lowPair, i, err := getMaxMinElev(hsSc, i, nLines, 0, 80, 8, 1)
	if err != nil {
		return highLowPairs, i, errors.Wrap(err, 0) 
	}
	highLowPairs[1] = lowPair

	return highLowPairs, i, nil
}

func stringtoFloat(s string) (float64, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed != "" {
		val, err := parseFloat(trimmed, 64)
		if err != nil {
			return 0, errors.Wrap(err, 0)
		}
		return val, nil
	}
	return 0, nil
}

// Extract data from Culvert Groups in HEC-RAS 1D Culverts,
// 1D Inline Structures, and Connections
func getConduits(line string, single bool) (conduits, error) {
	lineData := strings.Split(rightofEquals(line), ",")
	conduit := conduits{}

	if single {
		conduit.NumBarrels = 1
		conduit.Name = strings.TrimSpace(lineData[13])

	} else {
		numbarrels, err := strconv.Atoi(strings.TrimSpace(lineData[11]))
		if err != nil {
			return conduit, errors.Wrap(err, 0) 

		}
		conduit.NumBarrels = numbarrels
		conduit.Name = strings.TrimSpace(lineData[12])
	}

	shapeID, err := strconv.Atoi(strings.TrimSpace(lineData[0]))
	if err != nil {
		return conduit, errors.Wrap(err, 0) 

	}
	conduit.Shape = conduitShapes[shapeID]

	rise, err := stringtoFloat(lineData[1])
	if err != nil {
		return conduit, errors.Wrap(err, 0) 

	}
	conduit.Rise = rise

	span, err := stringtoFloat(lineData[2])
	if err != nil {
		return conduit, errors.Wrap(err, 0) 

	}
	conduit.Span = span

	length, err := stringtoFloat(lineData[3])
	if err != nil {
		return conduit, errors.Wrap(err, 0) 

	}
	conduit.Length = length

	mannings, err := stringtoFloat(lineData[4])
	if err != nil {
		return conduit, errors.Wrap(err, 0) 

	}
	conduit.ManningsN = mannings

	return conduit, nil
}

// Extract data from HEC-RAS 1D Culverts
func getCulvertData(hsSc *bufio.Scanner, i int, lineData []string) (culverts, int, error) {
	culvert := culverts{}

	station, err := parseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return culvert, i, errors.Wrap(err, 0) 
	}
	culvert.Station = station

	for hsSc.Scan() {
		i++
		line := hsSc.Text()
		switch {
		case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
			var description string
			description, i, err = getDescription(hsSc, i, "END DESCRIPTION:")
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.Description += description

		case strings.HasPrefix(line, "Node Name="):
			culvert.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			i++
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := parseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.DeckWidth = deckWidth

			var upHighLowPair [2]maxMinPairs
			upHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[4], 80, 8)
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.UpHighChord = upHighLowPair[0]
			culvert.UpLowChord = upHighLowPair[1]

			var downHighLowPair [2]maxMinPairs
			downHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[5], 80, 8)
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.DownHighChord = downHighLowPair[0]
			culvert.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Culvert="):
			conduit, err := getConduits(line, true)
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "Multiple Barrel Culv="):
			conduit, err := getConduits(line, false)
			if err != nil {
				return culvert, i, errors.Wrap(err, 0) 
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "Type RM Length L Ch R ="):
			return culvert, i, nil

		case strings.HasPrefix(line, "River Reach="):
			return culvert, i, nil
		}
	}
	return culvert, i, nil
}

// Extract data from 1D Bridges
func getBridgeData(hsSc *bufio.Scanner, i int, lineData []string) (bridges, int, error) {
	bridge := bridges{}

	station, err := parseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return bridge, i, errors.Wrap(err, 0) 
	}
	bridge.Station = station

	for hsSc.Scan() {
		i++
		line := hsSc.Text()
		switch {
		case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
			var description string
			description, i, err = getDescription(hsSc, i, "END DESCRIPTION:")
			if err != nil {
				return bridge, i, errors.Wrap(err, 0) 
			}
			bridge.Description += description

		case strings.HasPrefix(line, "Node Name="):
			bridge.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			i++
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := parseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return bridge, i, errors.Wrap(err, 0) 
			}
			bridge.DeckWidth = deckWidth

			var upHighLowPair [2]maxMinPairs
			upHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[4], 80, 8)
			if err != nil {
				return bridge, i, errors.Wrap(err, 0) 
			}
			bridge.UpHighChord = upHighLowPair[0]
			bridge.UpLowChord = upHighLowPair[1]

			var downHighLowPair [2]maxMinPairs
			downHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[5], 80, 8)
			if err != nil {
				return bridge, i, errors.Wrap(err, 0) 
			}
			bridge.DownHighChord = downHighLowPair[0]
			bridge.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Pier Skew"):
			bridge.NumPiers++

		case strings.HasPrefix(line, "Type RM Length L Ch R ="):
			return bridge, i, nil

		case strings.HasPrefix(line, "River Reach="):
			return bridge, i, nil
		}
	}
	return bridge, i, nil
}

// Extract data from Gates Groups in 1D Inline Structures and Connections
func getGates(nextLine string) (gates, error) {
	gate := gates{}

	nextLineData := strings.Split(nextLine, ",")

	gate.Name = strings.TrimSpace(nextLineData[0])

	width, err := stringtoFloat(nextLineData[1])
	if err != nil {
		return gate, errors.Wrap(err, 0) 

	}
	gate.Width = width

	height, err := stringtoFloat(nextLineData[2])
	if err != nil {
		return gate, errors.Wrap(err, 0) 

	}
	gate.Height = height

	numopenings, err := strconv.Atoi(strings.TrimSpace(nextLineData[13]))
	if err != nil {
		return gate, errors.Wrap(err, 0) 

	}
	gate.NumOpenings = numopenings

	return gate, nil
}

// Extract data from Inline Structures
func getWeirData(rm *RasModel, fn string, i int) (weirs, error) {
	weir := weirs{}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return weir, errors.Wrap(err, 0) 

	}
	defer f.Close()

	wSc := bufio.NewScanner(f)

	wi := 0
	for wSc.Scan() {
		wi++
		if wi == i {
			lineData := strings.Split(rightofEquals(wSc.Text()), ",")
			station, err := parseFloat(strings.TrimSpace(lineData[1]), 64)
			if err != nil {
				return weir, errors.Wrap(err, 0) 

			}
			weir.Station = station
		} else if wi > i {
			line := wSc.Text()
			switch {
			case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
				description, _, err := getDescription(wSc, 0, "END DESCRIPTION:")
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				weir.Description += description

			case strings.HasPrefix(line, "Node Name="):
				weir.Name = rightofEquals(line)

			case strings.HasPrefix(line, "#Inline Weir SE="):
				nElev, err := strconv.Atoi(strings.TrimSpace(rightofEquals(line)))
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				nLines := numberofLines(nElev*2, 80, 8)

				elev, _, err := getMaxMinElev(wSc, 0, nLines, 0, 80, 8, 2)
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				weir.WeirElev = elev

			case strings.HasPrefix(line, "IW Dist,WD"):
				wSc.Scan()
				nextLineData := strings.Split(wSc.Text(), ",")
				weirWidth, err := parseFloat(strings.TrimSpace(nextLineData[1]), 64)
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				weir.WeirWidth = weirWidth

			case strings.HasPrefix(line, "IW Gate Name"):
				wSc.Scan()
				gate, err := getGates(wSc.Text())
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				weir.Gates = append(weir.Gates, gate)
				weir.NumGates++

			case strings.HasPrefix(line, "IW Culv="):
				conduit, err := getConduits(line, false)
				if err != nil {
					return weir, errors.Wrap(err, 0) 

				}
				weir.Conduits = append(weir.Conduits, conduit)
				weir.NumConduits++

			case strings.HasPrefix(line, "Type RM Length L Ch R ="):
				return weir, nil

			case strings.HasPrefix(line, "River Reach="):
				return weir, nil
			}
		}
	}
	return weir, nil
}

// Extract all data from 1D Bridges, Culverts, and Inline Structures
func getHydraulicStructureData(rm *RasModel, fn string, idx int) (hydraulicStructures, error) {
	structures := hydraulicStructures{}
	bData := bridgeData{}
	cData := culvertData{}
	wData := weirData{}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return structures, errors.Wrap(err, 0) 

	}
	defer f.Close()

	hsSc := bufio.NewScanner(f)

	i := 0
	for hsSc.Scan() {
		i++
		if i == idx {
			riverReach := strings.Split(rightofEquals(hsSc.Text()), ",")
			structures.River = strings.TrimSpace(riverReach[0])
			structures.Reach = strings.TrimSpace(riverReach[1])
		} else if i > idx {
			line := hsSc.Text()
			if strings.HasPrefix(line, "Type RM Length L Ch R =") {
				data := strings.Split(rightofEquals(line), ",")
				structureType, err := strconv.Atoi(strings.TrimSpace(data[0]))
				if err != nil {
					return structures, errors.Wrap(err, 0) 

				}
				switch structureType {
				case 1:
					structures.NumXS++

				case 2:
					var culvert culverts
					culvert, i, err = getCulvertData(hsSc, i, data)
					if err != nil {
						return structures, errors.Wrap(err, 0) 

					}
					cData.Culverts = append(cData.Culverts, culvert)
					cData.NumCulverts++

				case 3:
					var bridge bridges
					bridge, i, err = getBridgeData(hsSc, i, data)
					if err != nil {
						return structures, errors.Wrap(err, 0) 

					}
					bData.Bridges = append(bData.Bridges, bridge)
					bData.NumBridges++

				case 5:
					weir, err := getWeirData(rm, fn, i)
					if err != nil {
						return structures, errors.Wrap(err, 0) 

					}
					wData.Weirs = append(wData.Weirs, weir)
					wData.NumWeirs++
				}
			}
			if strings.HasPrefix(line, "River Reach=") {
				structures.CulvertData = cData
				structures.BridgeData = bData
				structures.WeirData = wData
				return structures, nil
			}
		}
	}
	structures.CulvertData = cData
	structures.BridgeData = bData
	structures.WeirData = wData

	return structures, nil
}								