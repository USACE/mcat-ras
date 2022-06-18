// Structs and functions used to parse unsteady flow files.

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

// These prefixes are used to determine the beginning and end of HEC-RAS elements
var unsteadyElementsPrefix = [...]string{
	"Flow Title",
	"Program Version",
	"Boundary Location",
}

// Unsteady Data
type UnsteadyData struct {
	FlowTitle          string
	ProgramVersion     string
	InitialConditions  interface{} // to be implemented
	BoundaryConditions UnsteadyBoundaryConditions
	MeterologicalData  interface{} // to be implemented
	ObservedData       interface{} // to be implemented // added in version 6.2
}

// Unsteady Boundary Conditions
type UnsteadyBoundaryConditions struct {
	// There can be many boundary conditions for the same element
	Reaches     map[string][]BoundaryCondition
	Areas       map[string][]BoundaryCondition
	Connections map[string][]BoundaryCondition
	// Only one boundary condition for each element in pumps
	PumpStations map[string]BoundaryCondition
}

// Rating Curve
type RatingCurve struct {
	Values  [][2]float64 `json:"values,omitempty"`
	UseDSS  bool         `json:"use_dss"`
	DSSFile string       `json:"dss_file,omitempty"`
	DSSPath string       `json:"dss_path,omitempty"`
}

// Hydrograph Data.
// Can be Flow, Stage, Precipitation, Uniform Lateral Inflow, Lateral Inflow, Ground Water Interflow, or Gate Opening Hydrograph.
type Hydrograph struct {
	TimeInterval       string      `json:"time_interval,omitempty"`
	EndRS              string      `json:"flow_distribution_last_RS,omitempty"` // flow will be distributed from RS to EndRS. Valid for Reaches with Uniform Lateral Inflow or Groundwater Interflow
	Values             interface{} `json:"values,omitempty"`
	UseDSS             bool        `json:"use_dss"`
	DSSFile            string      `json:"dss_file,omitempty"`
	DSSPath            string      `json:"dss_path,omitempty"`
	UseFixedStart      bool        `json:"fixed_start"`
	FixedStartDateTime *DateTime   `json:"fixed_start_date_time,omitempty"` // pointer to have zero value, so that omitempty can work
}

type DateTime struct {
	Date  string `json:"date,omitempty"`
	Hours string `json:"hours,omitempty"` // should not be int/float or else 0015 hours will become 15 hours
}

// Parse Unsteady Boundary Condition's header.
func parseUnsteadyBCHeader(line string) (parentType string, parent string, flowEndRS string, bc BoundaryCondition, err error) {
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

// Get Rating Curve Boundary Condition Data
// Returns at EOF or if new Unsteady element is encountered
func getRatingCurveData(sc *bufio.Scanner) (rc RatingCurve, skipScan bool, err error) {

	series, innerErr := getDataPairsfromTextBlock("Rating Curve", sc, 80, 8)

	if innerErr != nil {
		return rc, false, innerErr
	}
	rc.Values = series

	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, unsteadyElementsPrefix[:]) {
			return rc, true, nil
		}

		switch loe {
		case "Use DSS":
			if rightofEquals(line) == "True" {
				rc.UseDSS = true
			}
		case "DSS File":
			rc.DSSFile = strings.TrimSpace(rightofEquals(sc.Text()))
		case "DSS Path":
			rc.DSSPath = strings.TrimSpace(rightofEquals(sc.Text()))
		}
	}
	return
}

// Get Hydrograph Data of a Boundary Condition
// Returns at EOF or if new Unsteady element is encountered
func getHydrographData(sc *bufio.Scanner, hydrographType string, pairedData bool, flowEndRS string) (hg Hydrograph, skipScan bool, err error) {

	if flowEndRS != "" {
		hg.EndRS = flowEndRS
	}

	if pairedData { // Stage and Flow Hydrograph or IB Stage and Flow
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
		if numVals != 0 {
			series, innerErr := seriesFromTextBlock(sc, numVals, 80, 8)

			if innerErr != nil {
				return hg, false, innerErr
			}
			hg.Values = series
		}
	}
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, unsteadyElementsPrefix[:]) {
			return hg, true, nil
		}

		switch loe {
		case "Use DSS":
			if rightofEquals(line) == "True" {
				hg.UseDSS = true
			}
		case "DSS File":
			hg.DSSFile = strings.TrimSpace(rightofEquals(sc.Text()))
		case "DSS Path":
			hg.DSSPath = strings.TrimSpace(rightofEquals(sc.Text()))
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
	return
}

// Get T. S. Gate Openings data
// Returns at EOF or if new Unsteady element is encountered
func getGateData(sc *bufio.Scanner) (gates map[string]*Hydrograph, skipScan bool, err error) {
	gates = make(map[string]*Hydrograph)
	var hg *Hydrograph

	eof := false
	for !eof {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, unsteadyElementsPrefix[:]) {
			skipScan = true
			return
		}

		switch loe {
		case "Gate Name":
			gateName := strings.TrimSpace(rightofEquals(line))
			// when new Gate starts, create a new variable to assign data to
			hg = &Hydrograph{}
			gates[gateName] = hg
		case "Gate Use DSS":
			if rightofEquals(line) == "True" {
				hg.UseDSS = true
			}
		case "Gate DSS File":
			hg.DSSFile = strings.TrimSpace(rightofEquals(sc.Text()))
		case "Gate DSS Path":
			hg.DSSPath = strings.TrimSpace(rightofEquals(sc.Text()))
		case "Gate Time Interval":
			hg.TimeInterval = rightofEquals(sc.Text())
		case "Gate Use Fixed Start Time":
			ufs := strings.TrimSpace(rightofEquals(line))
			if ufs == "True" {
				hg.UseFixedStart = true
			}
		case "Gate Fixed Start Date/Time":
			fsdt := strings.Split(rightofEquals(line), ",")
			if len(fsdt[0]) > 0 {
				hg.FixedStartDateTime = &DateTime{}
				hg.FixedStartDateTime.Date = fsdt[0]
				hg.FixedStartDateTime.Hours = fsdt[1]
			}
		case "Gate Openings":
			numValues, innerErr := strconv.Atoi(strings.TrimSpace(rightofEquals(sc.Text())))
			if innerErr != nil {
				return gates, false, innerErr
			}
			if numValues != 0 {
				data, innerErr := seriesFromTextBlock(sc, numValues, 80, 8)
				if innerErr != nil {
					return gates, false, innerErr
				}
				hg.Values = data
			}
		}
		eof = !sc.Scan()
	}
	return
}

// Get Boundary Condition's data.
// Advances the given scanner.
// Returns if new RAS element is encountered or all necessary data is obtained.
func getBoundaryCondition(sc *bufio.Scanner) (parentType string, parent string, bc BoundaryCondition, skipScan bool, err error) {
	// either Reaches, Connections, Areas, or Pump Stations
	// e.g. name of the river - reach, or name of Storage Area

	// Get Parent, Name, and Location of Boundary Condition
	parentType, parent, flowEndRS, bc, err := parseUnsteadyBCHeader(sc.Text())
	if err != nil {
		return
	}

	timeInterval := ""
	// Get type and data of boundary condition
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)
		if stringInSlice(loe, unsteadyElementsPrefix[:]) {
			if bc.Type == "" {
				bc.Type = "Unknown Type"
			}
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
		case "Interval":
			timeInterval = rightofEquals(sc.Text())
		case "Flow Hydrograph", "Precipitation Hydrograph", "Uniform Lateral Inflow Hydrograph", "Lateral Inflow Hydrograph", "Ground Water Interflow", "Stage Hydrograph":
			if loe == "Precipitation Hydrograph" {
				bc.Type = "Precipitation"
			} else {
				bc.Type = loe
			}
			hg, ss, innerErr := getHydrographData(sc, loe, false, flowEndRS)
			hg.TimeInterval = timeInterval
			skipScan = ss
			if innerErr != nil {
				err = innerErr
				return
			}
			bc.Data = hg
			return

		case "Stage and Flow Hydrograph", "Observed Stage and Flow Hydrograph":
			if loe == "Observed Stage and Flow Hydrograph" {
				bc.Type = "IB Stage and Flow Hydrograph"
			} else {
				bc.Type = loe
			}
			hg, ss, innerErr := getHydrographData(sc, loe, true, flowEndRS)
			hg.TimeInterval = timeInterval
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

		case "Gate Name": // Keyword for T.S Gate Openings
			gate, ss, innerErr := getGateData(sc)
			skipScan = ss
			if innerErr != nil {
				err = innerErr
				return
			}

			bc.Data = gate
			bc.Type = "T. S. Gate Openings"
			return

		case "Rule Operation", "Rule Expression": // both are keywords for Rules BC
			bc.Type = "Rules"
			bc.Data = "Not Implemented"
			return

		case "Elev Controlled Gate", "Navigation Dam":
			bc.Type = loe
			bc.Data = "Not Implemented"
			return

		}
	}

	return
}

// Get Forcing Data from unsteady flow file.
func getUnsteadyData(fd *ForcingData, fs filestore.FileStore, flowFilePath string) error {
	flowFileName := filepath.Base(flowFilePath)
	ud := UnsteadyData{
		InitialConditions: "Not Implemented",
		BoundaryConditions: UnsteadyBoundaryConditions{
			Reaches:      make(map[string][]BoundaryCondition),
			Areas:        make(map[string][]BoundaryCondition),
			Connections:  make(map[string][]BoundaryCondition),
			PumpStations: make(map[string]BoundaryCondition), // Unlike other features a pump cannot have multiple boundary conditions
		},
		MeterologicalData: "Not Implemented",
		ObservedData:      "Not Implemented",
	}

	file, err := fs.GetObject(flowFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	if err := sc.Err(); err != nil {
		return err
	}

	eof := !sc.Scan()
	for !eof {
		skipScan := false
		line := sc.Text()
		loe := leftofEquals(line)

		switch loe {
		case "Flow Title":
			ud.FlowTitle = strings.TrimSpace(rightofEquals(line))
		case "Program Version":
			ud.ProgramVersion = strings.TrimSpace(rightofEquals(line))
		case "Boundary Location":
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
