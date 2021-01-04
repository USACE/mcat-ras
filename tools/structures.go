package tools

import (
	"bufio"
	"math"
	"strconv"
	"strings"
)

type hydraulicStructures struct {
	River       string     `json:"River Name"`
	Reach       string     `json:"Reach Name"`
	NumXS       int        `json:"Num CrossSections"`
	NumCulverts int        `json:"Num Culverts"`
	BridgeData  bridgeData `json:"Bridges"`
	NumInlines  int        `json:"Num Inlines"`
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
	DownLowChord  chordPairs `json:"Downstream Max Chord"`
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

func datafromTextBlock(hsSc *bufio.Scanner, colWidth int, valueWidth int, nLines int, nSkipLines int) ([]float64, error) {
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

func getHighLowChord(hsSc *bufio.Scanner, nElevsText string, colWidth int, valueWidth int) ([2]chordPairs, error) {
	highLowPairs := [2]chordPairs{}

	nElevs, err := strconv.Atoi(strings.TrimSpace(nElevsText))
	if err != nil {
		return highLowPairs, err
	}

	nLines := numberofLines(nElevs, colWidth, valueWidth)

	elevHighChord, err := datafromTextBlock(hsSc, colWidth, valueWidth, nLines, nLines)
	if err != nil {
		return highLowPairs, err
	}

	maxHighCord, err := maxValue(elevHighChord)
	if err != nil {
		return highLowPairs, err
	}

	minHighCord, err := minValue(elevHighChord)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[0] = chordPairs{Max: maxHighCord, Min: minHighCord}

	elevLowChord, err := datafromTextBlock(hsSc, 80, 8, nLines, 0)
	if err != nil {
		return highLowPairs, err
	}

	maxLowCord, err := maxValue(elevLowChord)
	if err != nil {
		return highLowPairs, err
	}

	minLowCord, err := minValue(elevLowChord)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[1] = chordPairs{Max: maxLowCord, Min: minLowCord}
	return highLowPairs, nil
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

		case strings.HasPrefix(line, "BR Coef"):
			return bridge, err
		}
	}
	return bridge, nil
}

func getHydraulicStructureData(rm *RasModel, fn string, idx int) (hydraulicStructures, error) {
	structures := hydraulicStructures{}
	bData := bridgeData{}

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
					structures.NumCulverts++

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
	structures.BridgeData = bData

	return structures, nil
}
