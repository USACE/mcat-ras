package tools

import (
	"bufio"
	"math"
	"strconv"
	"strings"
)

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

type hydraulicStructures struct {
	River       string      `json:"River Name"`
	Reach       string      `json:"Reach Name"`
	NumXS       int         `json:"Num CrossSections"`
	CulvertData culvertData `json:"Culverts"`
	BridgeData  bridgeData  `json:"Bridges"`
	NumInlines  int         `json:"Num Inlines"`
}

type culvertData struct {
	NumCulverts int        `json:"Num Culverts"`
	Culverts    []culverts `json:"Culverts"`
}

type culverts struct {
	Name          string
	Station       float64
	Description   string
	DeckWidth     float64    `json:"Deck Width"`
	UpHighChord   chordPairs `json:"Upstream High Chord"`
	UpLowChord    chordPairs `json:"Upstream Low Chord"`
	DownHighChord chordPairs `json:"Downstream High Chord"`
	DownLowChord  chordPairs `json:"Downstream Low Chord"`
	NumConduits   int        `json:"Num Culvert Conduits"`
	Conduits      []conduits `json:"Culvert Conduits"`
}

type conduits struct {
	NumBarrels int    `json:"Num Barrels"`
	Group      string `json:"Culvert Group"`
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

type bridges struct {
	Name          string
	Station       float64
	Description   string
	DeckWidth     float64    `json:"Deck Width"`
	UpHighChord   chordPairs `json:"Upstream High Chord"`
	UpLowChord    chordPairs `json:"Upstream Low Chord"`
	DownHighChord chordPairs `json:"Downstream High Chord"`
	DownLowChord  chordPairs `json:"Downstream Low Chord"`
	NumPiers      int        `json:"Num Piers"`
}

type chordPairs struct {
	Max float64
	Min float64
}

func numberofLines(nValues int, colWidth int, valueWidth int) int {
	nLines := math.Ceil(float64(nValues) / (float64(colWidth) / float64(valueWidth)))
	return int(nLines)
}

func datafromTextBlock(hsSc *bufio.Scanner, nLines int, nSkipLines int, colWidth int, valueWidth int) ([]float64, error) {
	values := []float64{}
	nSkipped := 0
	nProcessed := 0
out:
	for hsSc.Scan() {
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
					val, err := strconv.ParseFloat(sVal, 64)
					if err != nil {
						return values, err
					}
					values = append(values, val)
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
	return values, nil
}

func getChord(hsSc *bufio.Scanner, nLines int, nSkipLines int, colWidth int, valueWidth int) (chordPairs, error) {
	pair := chordPairs{}

	elevChord, err := datafromTextBlock(hsSc, nLines, nSkipLines, colWidth, valueWidth)

	if err != nil {
		return pair, err
	}

	if len(elevChord) == 0 {
		return pair, nil
	}

	maxChord, err := maxValue(elevChord)
	if err != nil {
		return pair, err
	}

	minChord, err := minValue(elevChord)
	if err != nil {
		return pair, err
	}

	pair = chordPairs{Max: maxChord, Min: minChord}
	return pair, nil
}

func getHighLowChord(hsSc *bufio.Scanner, nElevText string, colWidth int, valueWidth int) ([2]chordPairs, error) {
	highLowPairs := [2]chordPairs{}

	nElev, err := strconv.Atoi(strings.TrimSpace(nElevText))
	if err != nil {
		return highLowPairs, err
	}

	nLines := numberofLines(nElev, 80, 8)

	highPair, err := getChord(hsSc, nLines, nLines, 80, 8)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[0] = highPair

	lowPair, err := getChord(hsSc, nLines, 0, 80, 8)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[1] = lowPair

	return highLowPairs, nil
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

func getConduits(line string, single bool, shapeMap map[int]string) (conduits, error) {
	lineData := strings.Split(rightofEquals(line), ",")
	conduit := conduits{}

	if single {
		conduit.NumBarrels = 1
		conduit.Group = strings.TrimSpace(lineData[13])

	} else {
		numbarrels, err := strconv.Atoi(strings.TrimSpace(lineData[11]))
		if err != nil {
			return conduit, err
		}
		conduit.NumBarrels = numbarrels
		conduit.Group = strings.TrimSpace(lineData[12])
	}

	shapeID, err := strconv.Atoi(strings.TrimSpace(lineData[0]))
	if err != nil {
		return conduit, err
	}
	conduit.Shape = shapeMap[shapeID]

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

func getCulvertData(hsSc *bufio.Scanner, lineData []string, shapeMap map[int]string) (culverts, error) {
	culvert := culverts{}

	station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return culvert, err
	}
	culvert.Station = station

	for hsSc.Scan() {
		line := hsSc.Text()
		switch {
		case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
			description, _, err := getDescription(hsSc, 0, "END DESCRIPTION:")
			if err != nil {
				return culvert, err
			}
			culvert.Description += description

		case strings.HasPrefix(line, "Node Name="):
			culvert.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return culvert, err
			}
			culvert.DeckWidth = deckWidth

			upHighLowPair, err := getHighLowChord(hsSc, nextLineData[4], 80, 8)
			if err != nil {
				return culvert, err
			}
			culvert.UpHighChord = upHighLowPair[0]
			culvert.UpLowChord = upHighLowPair[1]

			downHighLowPair, err := getHighLowChord(hsSc, nextLineData[5], 80, 8)
			if err != nil {
				return culvert, err
			}
			culvert.DownHighChord = downHighLowPair[0]
			culvert.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Culvert="):
			conduit, err := getConduits(line, true, shapeMap)
			if err != nil {
				return culvert, err
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "Multiple Barrel Culv="):
			conduit, err := getConduits(line, false, shapeMap)
			if err != nil {
				return culvert, err
			}
			culvert.Conduits = append(culvert.Conduits, conduit)
			culvert.NumConduits++

		case strings.HasPrefix(line, "BC Design"):
			return culvert, err
		}
	}
	return culvert, nil
}

func getBridgeData(hsSc *bufio.Scanner, lineData []string) (bridges, error) {
	bridge := bridges{}

	station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return bridge, err
	}
	bridge.Station = station

	for hsSc.Scan() {
		line := hsSc.Text()
		switch {
		case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
			description, _, err := getDescription(hsSc, 0, "END DESCRIPTION:")
			if err != nil {
				return bridge, err
			}
			bridge.Description += description

		case strings.HasPrefix(line, "Node Name="):
			bridge.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return bridge, err
			}
			bridge.DeckWidth = deckWidth

			upHighLowPair, err := getHighLowChord(hsSc, nextLineData[4], 80, 8)
			if err != nil {
				return bridge, err
			}
			bridge.UpHighChord = upHighLowPair[0]
			bridge.UpLowChord = upHighLowPair[1]

			downHighLowPair, err := getHighLowChord(hsSc, nextLineData[5], 80, 8)
			if err != nil {
				return bridge, err
			}
			bridge.DownHighChord = downHighLowPair[0]
			bridge.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Pier Skew"):
			bridge.NumPiers++

		case strings.HasPrefix(line, "BC Design"):
			return bridge, err
		}
	}
	return bridge, nil
}

func getHydraulicStructureData(rm *RasModel, fn string, idx int) (hydraulicStructures, error) {
	structures := hydraulicStructures{}
	bData := bridgeData{}
	cData := culvertData{}

	newf, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return structures, nil
	}
	defer newf.Close()

	hsSc := bufio.NewScanner(newf)

	i := 0
	for hsSc.Scan() {
		if i == idx {
			riverReach := strings.Split(rightofEquals(hsSc.Text()), ",")
			structures.River = strings.TrimSpace(riverReach[0])
			structures.Reach = strings.TrimSpace(riverReach[1])
		} else if i > idx {
			line := hsSc.Text()
			if strings.HasPrefix(line, "River Reach=") {
				structures.CulvertData = cData
				structures.BridgeData = bData
				return structures, nil
			}
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
					culvert, err := getCulvertData(hsSc, data, conduitShapes)
					if err != nil {
						return structures, err
					}
					cData.Culverts = append(cData.Culverts, culvert)
					cData.NumCulverts++

				case 3:
					bridge, err := getBridgeData(hsSc, data)
					if err != nil {
						return structures, err
					}
					bData.Bridges = append(bData.Bridges, bridge)
					bData.NumBridges++

				case 5:
					structures.NumInlines++

				}
			}
		}
		i++
	}
	structures.CulvertData = cData
	structures.BridgeData = bData

	return structures, nil
}
