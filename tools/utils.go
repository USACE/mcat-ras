package tools

import (
	"bufio"
	"math"
	"strconv"
	"strings"

	"github.com/go-errors/errors" // warning: replaces standard errors
)

func maxValue(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0.0, errors.New("Cannot detect a maximum value in an empty slice")
	}

	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	return max, nil
}

func minValue(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0.0, errors.New("Cannot detect a minimum value in an empty slice")
	}

	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}

	return min, nil
}

func rightofEquals(line string) string {
	return strings.TrimSpace(strings.Split(line, "=")[1])
}
func leftofEquals(line string) string {
	return strings.TrimSpace(strings.Split(line, "=")[0])
}

func getDescription(sc *bufio.Scanner, idx int, endLine string) (string, int, error) {
	description := ""
	nLines := 0
	for sc.Scan() {
		idx++
		line := sc.Text()
		if strings.HasPrefix(line, endLine) {
			return description, idx, nil
		}
		if line != "" {
			if nLines > 0 {
				description += "\n"
			}
			description += line
			nLines++
		}
	}
	return description, idx, nil
}

func getDescriptionConnections(sc *bufio.Scanner, endLine string) (string, error) {
	description := rightofEquals(sc.Text())
	nLines := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, endLine) {
			return description, nil
		}
		if line != "" {
			if nLines > 0 {
				description += "\n"
			}
			description += line
			nLines++
		}
	}
	return description, nil
}

func stringInSlice(val string, s []string) bool {
	for i := range s {
		if s[i] == val {
			return true
		}
	}
	return false
}

func parseFloat(s string, bitSize int) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, bitSize)
}

func numberofLines(nValues int, colWidth int, valueWidth int) int {
	nLines := math.Ceil(float64(nValues) / (float64(colWidth) / float64(valueWidth)))
	return int(nLines)
}

// Get series from HEC-RAS Text block that contains series e.g. Flow Hydrograph
func seriesFromTextBlock(sc *bufio.Scanner, nValues int, colWidth int, valueWidth int) ([]float64, error) {
	series := make([]float64, nValues)

	textValues, err := parseSeriesTextBlock(sc, nValues, colWidth, valueWidth)
	if err != nil {
		return series, err
	}
	for i, val := range textValues {
		floatVal, err := parseFloat(val, 64)
		if err != nil {
			return series, errors.Wrap(err, 0)
		}
		series[i] = floatVal
	}

	return series, nil
}

// Returns a series of strings (rather than floats)
// Can check for empty entries rather than set to 0
func parseSeriesTextBlock(sc *bufio.Scanner, nValues int, colWidth int, valueWidth int) ([]string, error) {
	series := make([]string, nValues)
	i := 0
out:
	for sc.Scan() {
		line := sc.Text()
		for s := 0; s < colWidth; s += valueWidth {
			if len(line) > s {
				val := strings.TrimSpace(line[s : s+valueWidth])

				series[i] = val
				i++
				if i == nValues {
					break out
				}
			} else {
				break
			}
		}
	}
	return series, nil
}

// Get pairs' series from HEC-RAS Text block that contains paired series e.g. Stage/Flow, X/Y
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

// Returns a paired data series.
// Gets data from next Paired Data Block encountered in HEC-RAS files.
// Returns at the successful parsing or at the end of the file.
func getDataPairsfromTextBlock(nDataPairsLine string, sc *bufio.Scanner, colWidth int, valueWidth int) ([][2]float64, error) {
	pairs := [][2]float64{}
	for {
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
		if !sc.Scan() {
			break
		}

		// to do: there should be a check here to see it has not enocuntered new element
	}
	return pairs, nil
}
