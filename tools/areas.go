package tools

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-errors/errors" // warning: replaces standard errors
)

type StorageArea struct {
	NumBCLines int      `json:"Num BC Lines"`
	BCLines    []string `json:"BC Lines"`
}

type TwoDArea struct {
	NumCells   int      `json:"Num Mesh Cells"`
	NumBCLines int      `json:"Num BC Lines"`
	BCLines    []string `json:"BC Lines"`
}

// Extract Storage and 2D Areas Data
func getAreasData(rm *RasModel, fn string, i int) (string, interface{}, error) {
	var name, is2D string
	var numCells int

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return "", nil, errors.Wrap(err, 0)
	}
	defer f.Close()

	aSc := bufio.NewScanner(f)
	var ai int

areaLoop:
	for aSc.Scan() {
		ai++
		if ai == i {
			lineData := strings.Split(rightofEquals(aSc.Text()), ",")
			name = strings.TrimSpace(lineData[0])

		} else if ai > i {
			line := aSc.Text()
			switch {

			case strings.HasPrefix(line, "Storage Area Is2D="):
				is2D = rightofEquals(line)
				if is2D != "0" && is2D != "-1" {
					return "", nil, errors.New(fmt.Sprintf("Cannot determine if area is storage area or 2D area at line '%v' of %v", line, fn))
				}

			case strings.HasPrefix(line, "Storage Area 2D Points="):
				numCells, err = strconv.Atoi(rightofEquals(line))
				if err != nil {
					return "", nil, errors.Wrap(err, 0)
				}

			case strings.HasPrefix(line, "2D Face Area "):
				break areaLoop

			case strings.HasPrefix(line, "Storage Area="):
				// guard to make sure new Storage Area don't overwrite previous values
				break areaLoop
			}
		}
	}
	if is2D == "0" {
		area := StorageArea{}
		return name, area, nil
	} else {
		area := TwoDArea{
			NumCells: numCells,
		}
		return name, area, nil
	}

	return "", nil, errors.New(fmt.Sprintf("Failed to parse storage area at geom file line number %v of %v", i, fn))
}

// Extract Boundary Condition Line Data
func getBCLineData(rm *RasModel, fn string, i int) (string, string, error) {
	var bc string

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return "", bc, errors.Wrap(err, 0)
	}
	defer f.Close()

	bcSc := bufio.NewScanner(f)
	var bci int

	for bcSc.Scan() {
		bci++
		if bci == i {
			bc = rightofEquals(bcSc.Text())

		} else if bci > i {
			line := bcSc.Text()
			switch {

			case strings.HasPrefix(line, "BC Line Storage Area="):
				area := rightofEquals(line)
				return area, bc, nil

			case strings.HasPrefix(line, "BC Line Text Position="):
				return "", bc, errors.New(fmt.Sprintf("Failed to parse BC Line at geom file line number %v of %v", i, fn))

			case strings.HasPrefix(line, "BC Line Name="):
				// returning error here because associated area is a must field
				return "", bc, errors.New(fmt.Sprintf("Failed to parse BC Line at geom file line number %v of %v", i, fn))

			}
		}
	}
	return "", bc, errors.New(fmt.Sprintf("Failed to parse BC Line at geom file line number %v of %v", i, fn))
}
