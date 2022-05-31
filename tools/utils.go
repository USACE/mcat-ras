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

// Get data series from HEC-RAS Text block
func seriesFromTextBlock(sc *bufio.Scanner, nValues int, colWidth int, valueWidth int) ([]float64, error) {
	series := []float64{}
out:
	for sc.Scan() {
		line := sc.Text()
		for s := 0; s < colWidth; {
			if len(line) > s {
				val, err := parseFloat(strings.TrimSpace(line[s:s+valueWidth]), 64)
				if err != nil {
					return series, errors.Wrap(err, 0)
				}
				series = append(series, val)
				if len(series) == nValues {
					break out
				}
			} else {
				break
			}
			s += valueWidth
		}
	}
	return series, nil
}
