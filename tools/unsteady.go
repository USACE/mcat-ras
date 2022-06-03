package tools

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/USACE/filestore"
	"github.com/go-errors/errors"
)

// Unsteady Data
type UnsteadyData struct {
	// omitting empty here because null InitialConditions maybe assumed as no initial condition exist, while in fact it may or may not exist until implemented
	InitialConditions  interface{}                `json:",omitempty"` // to be implemented
	BoundaryConditions UnsteadyBoundaryConditions `json:",omitempty"`
	MeterologicalData  interface{}                `json:",omitempty"` // to be implemented
	ObservedData       interface{}                `json:",omitempty"` // to be implemented // added in version 6.2
}

// Unsteady Boundary Conditions
type UnsteadyBoundaryConditions struct {
	Reaches      map[string][]BoundaryCondition
	Areas        map[string][]BoundaryCondition
	Connections  map[string][]BoundaryCondition
	PumpStations map[string]BoundaryCondition
}

// Parse Boundary Condition's header.
func parseBCHeader(line string) (parentType string, parent string, flowEndRS string, bc BoundaryCondition, err error) {
	bcArray := strings.Split(rightofEquals(line), ",")
	if strings.TrimSpace(bcArray[0]) != "" {
		parent = fmt.Sprintf("%s - %s", strings.TrimSpace(bcArray[0]), strings.TrimSpace(bcArray[1]))
		parentType = "Reach"
		bc.RS = strings.TrimSpace(bcArray[2])
		flowEndRS = strings.TrimSpace(bcArray[3])
	} else if strings.TrimSpace(bcArray[4]) != "" {
		parent = strings.TrimSpace(bcArray[4])
		parentType = "Connection"
	} else if strings.TrimSpace(bcArray[5]) != "" {
		parent = strings.TrimSpace(bcArray[5])
		parentType = "Area"
		bc.BCLine = strings.TrimSpace(bcArray[7])
	} else if strings.TrimSpace(bcArray[6]) != "" {
		parent = strings.TrimSpace(bcArray[6])
		parentType = "PumpStation"
	}

	if parentType == "" {
		err = errors.Errorf("Cannot determine if Boundary Condition is for a Reach, Connection, Area, or Pump Station at line '%s'.", line)
		return
	}
	return
}

// Return a RatingCurve object, bool skipScan, error encountered
func getRatingCurveData(sc *bufio.Scanner) (RatingCurve, bool, error) {
	rc := RatingCurve{}

	series, innerErr := getDataPairsfromTextBlock("Rating Curve", sc, 80, 8)

	if innerErr != nil {
		return rc, false, innerErr
	}
	rc.Values = series

	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, forcingElementsPrefix[:]) {
			return rc, true, nil
		}

		if strings.HasPrefix(line, "Use DSS") {
			if rightofEquals(line) == "True" {
				rc.UseDSS = true
			}
			return rc, false, nil
		}
	}
	return rc, false, nil
}

// Return a Hydrograph object, bool skipScan, error encountered
func getHydrographData(sc *bufio.Scanner, hydrographType string, pairedData bool, flowEndRS string) (Hydrograph, bool, error) {
	hg := Hydrograph{}

	if flowEndRS != "" {
		hg.EndRS = flowEndRS
	}

	if pairedData { // Stage and Flow Hydrograph
		series, innerErr := getDataPairsfromTextBlock(hydrographType, sc, 80, 8)

		if innerErr != nil {
			return hg, false, innerErr
		}
		hg.Values = series
	} else {
		numVals, innerErr := strconv.Atoi(strings.TrimSpace(rightofEquals(sc.Text())))
		if innerErr != nil {
			return hg, false, innerErr
		}

		series, innerErr := seriesFromTextBlock(sc, numVals, 80, 8)

		if innerErr != nil {
			return hg, false, innerErr
		}
		hg.Values = series
	}

	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, forcingElementsPrefix[:]) {
			return hg, true, nil
		}

		switch loe {
		case "Use DSS":
			if rightofEquals(line) == "True" {
				hg.UseDSS = true
			}

		case "Use Fixed Start Time":
			ufs := strings.TrimSpace(rightofEquals(line))
			if ufs == "True" {
				hg.UseFixedStart = true
			}
		case "Fixed Start Date/Time":
			fsdt := strings.Split(rightofEquals(sc.Text()), ",")
			if len(fsdt[0]) > 0 {
				hg.FixedStartDateTime = &DateTime{}
				hg.FixedStartDateTime.Date = fsdt[0]
				hg.FixedStartDateTime.Hours = fsdt[1]
			}
		}
	}

	return hg, true, nil
}

// Get Boundary Condition's data.
// Advances the given scanner.
// Returns if new RAS element is encountered or all necessary data is obtained.
func getBoundaryCondition(sc *bufio.Scanner) (parentType string, parent string, bc BoundaryCondition, skipScan bool, err error) {
	// either Reaches, Connections, Areas, or Pump Stations
	// e.g. name of the river - reach, or name of Storage Area

	// Get Parent, Name, and Location of Boundary Condition
	parentType, parent, flowEndRS, bc, err := parseBCHeader(sc.Text())
	if err != nil {
		return
	}

	// Get type and data of boundary condition
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)
		if stringInSlice(loe, forcingElementsPrefix[:]) {
			skipScan = true // a new HEC RAS element has been encountered, skip next scan and return
			return
		}

		// findout type of BC
		switch loe {
		case "Friction Slope":
			bc.Type = "Normal Depth"
			slope, _ := parseFloat(strings.TrimSpace(strings.Split(rightofEquals(line), ",")[0]), 64)
			bc.Data = map[string]float64{"Friction Slope": slope}
			return
		// case "Interval":
		// 	hg.TimeInterval = rightofEquals(sc.Text())
		case "Flow Hydrograph", "Precipitation Hydrograph", "Uniform Lateral Inflow Hydrograph", "Lateral Inflow Hydrograph", "Ground Water Interflow", "Stage Hydrograph":
			bc.Type = loe
			hg, ss, innerErr := getHydrographData(sc, loe, false, flowEndRS)
			skipScan = ss
			if innerErr != nil {
				err = innerErr
				return
			}
			bc.Data = hg
			return

		case "Stage and Flow Hydrograph":
			bc.Type = loe
			hg, ss, innerErr := getHydrographData(sc, loe, true, flowEndRS)
			skipScan = ss

			if err != nil {
				err = innerErr
				return
			}
			bc.Data = hg
			return

		case "Rating Curve":
			rc, ss, innerErr := getRatingCurveData(sc)
			skipScan = ss

			if innerErr != nil {
				err = innerErr
				return
			}

			bc.Data = rc
			bc.Type = loe
			return
		}
	}

	return
}

// Get Forcing Data from unsteady flow file.
func getUnsteadyData(fd *ForcingData, fs filestore.FileStore, flowFilePath string) error {
	flowFileName := filepath.Base(flowFilePath)
	ud := UnsteadyData{
		BoundaryConditions: UnsteadyBoundaryConditions{
			Reaches:      make(map[string][]BoundaryCondition),
			Areas:        make(map[string][]BoundaryCondition),
			Connections:  make(map[string][]BoundaryCondition),
			PumpStations: make(map[string]BoundaryCondition), // Unlike other features a pump cannot have multiple boundary conditions
		},
	}

	file, err := fs.GetObject(flowFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	eof := !sc.Scan()
	if err := sc.Err(); err != nil {
		return err
	}

	for !eof {
		skipScan := false
		line := sc.Text()

		switch {
		case strings.HasPrefix(line, "Boundary Location="):
			parentType, parent, bc, ss, err := getBoundaryCondition(sc)
			skipScan = ss
			if err != nil {
				return errors.Wrap(err, 0)
			}

			switch parentType {
			case "Reach":
				ud.BoundaryConditions.Reaches[parent] = append(ud.BoundaryConditions.Reaches[parent], bc)
			case "Area":
				ud.BoundaryConditions.Areas[parent] = append(ud.BoundaryConditions.Areas[parent], bc)
			case "Connection":
				ud.BoundaryConditions.Connections[parent] = append(ud.BoundaryConditions.Reaches[parent], bc)
			case "PumpStation":
				ud.BoundaryConditions.PumpStations[parent] = bc
			}
		}

		// if a new RAS element is encountered during the functions call, scanning again will skip that element, therefore skip scan
		if !skipScan {
			eof = !sc.Scan()
			if err := sc.Err(); err != nil {
				return err
			}
		}
	}
	fd.Unsteady[flowFileName] = ud
	return nil
}
