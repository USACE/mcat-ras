package tools

import (
	"bufio"
	"errors"
	"math"
	"strconv"
	"strings"
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

// Store HEC-RAS 2D Connections
type connections2d struct {
	Name        string      `json:"Connection Name"`
	Description string      `json:"Connection Description"`
	UpSA        string      `json:"Connection Up SA"`
	DnSA        string      `json:"Connection Dn SA"`
	WeirWidth   float64     `json:"Weir Width"`
	WeirElev    maxMinPairs `json:"Weir Elevations"`
	NumGates    int         `json:"Num Gates"`
	Gates       []gates
	NumConduits int        `json:"Num Culvert Conduits"`
	Conduits    []conduits `json:"Culvert Conduits"`
}

type culvertData struct {
	NumCulverts int        `json:"Num Culverts"`
	Culverts    []culverts `json:"Culverts"`
}

// Store HEC-RAS 1D Culverts. This is different than culverts
// associated with inline structures or 2D connections
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

// Store Culvert Groups in HEC-RAS 1D Culverts, 1D Inline Structures, and 2D Connections
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

// Store Gates Groups in HEC-RAS 1D Inline Structures and 2D Connections
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
						val, err := strconv.ParseFloat(sVal, 64)
						if err != nil {
							return values, i, err
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
		return pair, i, err
	}

	if len(elevations) == 0 {
		return pair, i, nil
	}

	maxElev, err := maxValue(elevations)
	if err != nil {
		return pair, i, err
	}

	minElev, err := minValue(elevations)
	if err != nil {
		return pair, i, err
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
		return highLowPairs, i, err
	}

	nLines := numberofLines(nElev, 80, 8)

	highPair, i, err := getMaxMinElev(hsSc, i, nLines, nLines, 80, 8, 1)
	if err != nil {
		return highLowPairs, i, err
	}
	highLowPairs[0] = highPair

	lowPair, i, err := getMaxMinElev(hsSc, i, nLines, 0, 80, 8, 1)
	if err != nil {
		return highLowPairs, i, err
	}
	highLowPairs[1] = lowPair

	return highLowPairs, i, nil
}

func stringtoFloat(s string) (float64, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed != "" {
		val, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}
	return 0, nil
}

// Extract data from Culvert Groups in HEC-RAS 1D Culverts,
// 1D Inline Structures, and 2D Connections
func getConduits(line string, single bool) (conduits, error) {
	lineData := strings.Split(rightofEquals(line), ",")
	conduit := conduits{}

	if single {
		conduit.NumBarrels = 1
		conduit.Name = strings.TrimSpace(lineData[13])

	} else {
		numbarrels, err := strconv.Atoi(strings.TrimSpace(lineData[11]))
		if err != nil {
			return conduit, err
		}
		conduit.NumBarrels = numbarrels
		conduit.Name = strings.TrimSpace(lineData[12])
	}

	shapeID, err := strconv.Atoi(strings.TrimSpace(lineData[0]))
	if err != nil {
		return conduit, err
	}
	conduit.Shape = conduitShapes[shapeID]

	rise, err := stringtoFloat(lineData[1])
	if err != nil {
		return conduit, err
	}
	conduit.Rise = rise

	span, err := stringtoFloat(lineData[2])
	if err != nil {
		return conduit, err
	}
	conduit.Span = span

	length, err := stringtoFloat(lineData[3])
	if err != nil {
		return conduit, err
	}
	conduit.Length = length

	mannings, err := stringtoFloat(lineData[4])
	if err != nil {
		return conduit, err
	}
	conduit.ManningsN = mannings

	return conduit, nil
}

// Extract data from HEC-RAS 1D Culverts
func getCulvertData(hsSc *bufio.Scanner, i int, lineData []string) (culverts, int, error) {
	culvert := culverts{}

	station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return culvert, i, err
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
				return culvert, i, err
			}
			culvert.Description += description

		case strings.HasPrefix(line, "Node Name="):
			culvert.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			i++
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return culvert, i, err
			}
			culvert.DeckWidth = deckWidth

			var upHighLowPair [2]maxMinPairs
			upHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[4], 80, 8)
			if err != nil {
				return culvert, i, err
			}
			culvert.UpHighChord = upHighLowPair[0]
			culvert.UpLowChord = upHighLowPair[1]

			var downHighLowPair [2]maxMinPairs
			downHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[5], 80, 8)
			if err != nil {
				return culvert, i, err
			}
			culvert.DownHighChord = downHighLowPair[0]
			culvert.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Culvert="):
			conduit, err := getConduits(line, true)
			if err != nil {
				return culvert, i, err
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "Multiple Barrel Culv="):
			conduit, err := getConduits(line, false)
			if err != nil {
				return culvert, i, err
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "BC Design"):
			return culvert, i, nil

		case strings.HasPrefix(line, "Type RM Length L Ch R ="):
			return culvert, i, errors.New("Failed to terminate parsing of culvert at 'BC Design'")

		case strings.HasPrefix(line, "River Reach="):
			return culvert, i, errors.New("Failed to terminate parsing of culvert at 'BC Design'")
		}
	}
	return culvert, i, nil
}

// Extract data from 1D Bridges
func getBridgeData(hsSc *bufio.Scanner, i int, lineData []string) (bridges, int, error) {
	bridge := bridges{}

	station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return bridge, i, err
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
				return bridge, i, err
			}
			bridge.Description += description

		case strings.HasPrefix(line, "Node Name="):
			bridge.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			i++
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return bridge, i, err
			}
			bridge.DeckWidth = deckWidth

			var upHighLowPair [2]maxMinPairs
			upHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[4], 80, 8)
			if err != nil {
				return bridge, i, err
			}
			bridge.UpHighChord = upHighLowPair[0]
			bridge.UpLowChord = upHighLowPair[1]

			var downHighLowPair [2]maxMinPairs
			downHighLowPair, i, err = getHighLowChord(hsSc, i, nextLineData[5], 80, 8)
			if err != nil {
				return bridge, i, err
			}
			bridge.DownHighChord = downHighLowPair[0]
			bridge.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Pier Skew"):
			bridge.NumPiers++

		case strings.HasPrefix(line, "BC Design"):
			return bridge, i, nil

		case strings.HasPrefix(line, "Type RM Length L Ch R ="):
			return bridge, i, errors.New("Failed to terminate parsing of bridge at 'BC Design'")

		case strings.HasPrefix(line, "River Reach="):
			return bridge, i, errors.New("Failed to terminate parsing of bridge at 'BC Design'")
		}
	}
	return bridge, i, nil
}

// Extract data from Gates Groups in 1D Inline Structures and 2D Connections
func getGates(nextLine string) (gates, error) {
	gate := gates{}

	nextLineData := strings.Split(nextLine, ",")

	gate.Name = strings.TrimSpace(nextLineData[0])

	width, err := stringtoFloat(nextLineData[1])
	if err != nil {
		return gate, err
	}
	gate.Width = width

	height, err := stringtoFloat(nextLineData[2])
	if err != nil {
		return gate, err
	}
	gate.Height = height

	numopenings, err := strconv.Atoi(strings.TrimSpace(nextLineData[13]))
	if err != nil {
		return gate, err
	}
	gate.NumOpenings = numopenings

	return gate, nil
}

// Extract data from Inline Structures
func getWeirData(rm *RasModel, fn string, i int) (weirs, error) {
	weir := weirs{}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return weir, err
	}
	defer f.Close()

	wSc := bufio.NewScanner(f)

	wi := 0
	for wSc.Scan() {
		wi++
		if wi == i {
			lineData := strings.Split(rightofEquals(wSc.Text()), ",")
			station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
			if err != nil {
				return weir, err
			}
			weir.Station = station
		} else if wi > i {
			line := wSc.Text()
			switch {
			case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
				description, _, err := getDescription(wSc, 0, "END DESCRIPTION:")
				if err != nil {
					return weir, err
				}
				weir.Description += description

			case strings.HasPrefix(line, "Node Name="):
				weir.Name = rightofEquals(line)

			case strings.HasPrefix(line, "#Inline Weir SE="):
				nElev, err := strconv.Atoi(strings.TrimSpace(rightofEquals(line)))
				if err != nil {
					return weir, err
				}
				nLines := numberofLines(nElev*2, 80, 8)

				elev, _, err := getMaxMinElev(wSc, 0, nLines, 0, 80, 8, 2)
				if err != nil {
					return weir, err
				}
				weir.WeirElev = elev

			case strings.HasPrefix(line, "IW Dist,WD"):
				wSc.Scan()
				nextLineData := strings.Split(wSc.Text(), ",")
				weirWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[1]), 64)
				if err != nil {
					return weir, err
				}
				weir.WeirWidth = weirWidth

			case strings.HasPrefix(line, "IW Gate Name"):
				wSc.Scan()
				gate, err := getGates(wSc.Text())
				if err != nil {
					return weir, err
				}
				weir.Gates = append(weir.Gates, gate)
				weir.NumGates++

			case strings.HasPrefix(line, "IW Culv="):
				conduit, err := getConduits(line, false)
				if err != nil {
					return weir, err
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
		return structures, err
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
					return structures, err
				}
				switch structureType {
				case 1:
					structures.NumXS++

				case 2:
					var culvert culverts
					culvert, i, err = getCulvertData(hsSc, i, data)
					if err != nil {
						return structures, err
					}
					cData.Culverts = append(cData.Culverts, culvert)
					cData.NumCulverts++

				case 3:
					var bridge bridges
					bridge, i, err = getBridgeData(hsSc, i, data)
					if err != nil {
						return structures, err
					}
					bData.Bridges = append(bData.Bridges, bridge)
					bData.NumBridges++

				case 5:
					weir, err := getWeirData(rm, fn, i)
					if err != nil {
						return structures, err
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

// Extract data from 2D Connections
func getConnectionsData(rm *RasModel, fn string, i int) (connections2d, error) {
	connection := connections2d{}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return connection, err
	}
	defer f.Close()

	cSc := bufio.NewScanner(f)

	ci := 0
	for cSc.Scan() {
		ci++
		if ci == i {
			lineData := strings.Split(rightofEquals(cSc.Text()), ",")
			name := strings.TrimSpace(lineData[0])

			connection.Name = name
		} else if ci > i {
			line := cSc.Text()
			switch {
			case strings.HasPrefix(line, "Connection Desc="):
				description, _, err := getDescriptionConnections(cSc, 0, "Connection Line=")
				if err != nil {
					return connection, err
				}
				connection.Description += description

			case strings.HasPrefix(line, "Connection Up SA="):
				connection.UpSA = rightofEquals(line)

			case strings.HasPrefix(line, "Connection Dn SA="):
				connection.DnSA = rightofEquals(line)

			case strings.HasPrefix(line, "Conn Weir WD="):
				weirWidth, err := strconv.ParseFloat(strings.TrimSpace(rightofEquals(line)), 64)
				if err != nil {
					return connection, err
				}
				connection.WeirWidth = weirWidth

			case strings.HasPrefix(line, "Conn Weir SE="):
				nElev, err := strconv.Atoi(strings.TrimSpace(rightofEquals(line)))
				if err != nil {
					return connection, err
				}
				nLines := numberofLines(nElev*2, 80, 8)

				elev, _, err := getMaxMinElev(cSc, 0, nLines, 0, 80, 8, 2)
				if err != nil {
					return connection, err
				}
				connection.WeirElev = elev

			case strings.HasPrefix(line, "Conn Gate Name Wd,H,"):
				cSc.Scan()
				gate, err := getGates(cSc.Text())
				if err != nil {
					return connection, err
				}
				connection.Gates = append(connection.Gates, gate)
				connection.NumGates++

			case strings.HasPrefix(line, "Connection Culv="):
				conduit, err := getConduits(line, false)
				if err != nil {
					return connection, err
				}
				connection.Conduits = append(connection.Conduits, conduit)
				connection.NumConduits++

			case strings.HasPrefix(line, "Conn Outlet Rating Curve="):
				return connection, nil

			}
		}
	}
	return connection, nil
}
