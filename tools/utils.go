package tools

import (
	"bufio"
	"errors"
	"strings"
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

func getDescriptionConnections(sc *bufio.Scanner, idx int, endLine string) (string, int, error) {
	description := rightofEquals(sc.Text())
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
