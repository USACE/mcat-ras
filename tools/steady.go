package tools

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/USACE/filestore"
	"github.com/go-errors/errors"
)

// These prefixes are used to determine the beginning and end of HEC-RAS elements
var steadyElementsPrefix = [...]string{"River Rch & RM=", "Boundary for River Rch & Prof#="}

// map of steadyBoundaryConditionTypes
var bcTypeMapping = map[string]string{
	"0": "",
	"1": "Known WS",
	"2": "Critical Depth",
	"3": "Normal Depth",
	"4": "Rating Curve",
}

// Steady Data
type SteadyData struct {
	FlowTitle      string
	ProgramVersion string
	Profiles       []Profile
}

// Steady Flow Profile ...
type Profile struct {
	Name                  string
	BoundaryConditions    map[string]*map[string]BoundaryCondition
	Flows                 map[string][]RSFlow
	StorageAreaElevations []StoAreaElevation
}

// River Flow Data Pair...
type RSFlow struct {
	RS   string  `json:"river_station"`
	Flow float64 `json:"flow"`
}

// Storage Area Elevation Data Pair...
type StoAreaElevation struct {
	SorageArea string  `json:"storage_area"`
	Elevation  float64 `json:"elevation"`
}

// Get Number of Profiles
func getNameNumProfiles(file *io.ReadCloser) (numProf int, names []string, err error) {
	sc := bufio.NewScanner(*file)
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if loe == "Number of Profiles" {
			numProf, err = strconv.Atoi(strings.TrimSpace(rightofEquals(line)))
			if err != nil {
				return
			}
		} else if loe == "Number of Profiles" {
			names = strings.Split(rightofEquals(line), ",")
		}
		if numProf == len(names) {
			return
		}
	}
	return numProf, names, errors.Errorf("Couldn't find number of profiles")
}

// Get HEC RAS Flow Files Title and Program Version.
// Advances the given scanner.
// Returns only when new element is encountered.
func getFlowTitleVersion(sc *bufio.Scanner, elementsPrefix []string) (title string, version string, skipScan bool) {
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)

		if stringInSlice(loe, elementsPrefix) {
			skipScan = true // a new HEC RAS element has been encountered, skip next scan and return
			return
		}

		switch loe {
		case "Flow Title":
			title = strings.TrimSpace(rightofEquals(line))
		case "Program Version":
			version = strings.TrimSpace(rightofEquals(line))
		}
	}
	return
}

// Parse Boundary Condition's header.
func parseRFHeader(line string) (reach string, rs string, err error) {
	rfArray := strings.Split(rightofEquals(line), ",")
	if strings.TrimSpace(rfArray[0]) != "" {
		reach = fmt.Sprintf("%s - %s", strings.TrimSpace(rfArray[0]), strings.TrimSpace(rfArray[1]))
		rs = strings.TrimSpace(rfArray[2])
		return
	} else {
		err = errors.Errorf("Cannot determine River/Reach name at line '%s'.", line)
		return
	}
}

// Get Reach Flow.
// Advances the given scanner.
func getReachFlows(sc *bufio.Scanner, sd *SteadyData) error {

	// Get Name, and Location of reach
	reach, rs, err := parseRFHeader(sc.Text())
	if err != nil {
		return err
	}

	series, err := seriesFromTextBlock(sc, len(sd.Profiles), 80, 8)
	for index, element := range series {
		sd.Profiles[index].Flows[reach] = append(sd.Profiles[index].Flows[reach], RSFlow{rs, element})
	}
	return nil
}

// Parse Boundary Condition's header.
func parseSteadyBCHeader(line string) (reach string, profNum int, err error) {
	bcArray := strings.Split(rightofEquals(line), ",")
	if strings.TrimSpace(bcArray[0]) != "" {
		reach = fmt.Sprintf("%s - %s", strings.TrimSpace(bcArray[0]), strings.TrimSpace(bcArray[1]))
		profNum, err = strconv.Atoi(strings.TrimSpace(bcArray[2]))
		return
	} else {
		err = errors.Errorf("Cannot determine River/Reach name, profile number at line '%s'.", line)
		return
	}
}

// Get Boundary Condition's data.
// Advances the given scanner.
// Returns only when new RAS element is encountered
func getReachBCs(sc *bufio.Scanner, sd *SteadyData) (skipScan bool, err error) {

	// Get Reach and Profile Number of  Boundary Condition
	reach, profNum, err := parseSteadyBCHeader(sc.Text())
	if err != nil {
		return
	}

	bcs := map[string]BoundaryCondition{
		"Up": BoundaryCondition{},
		"Dn": BoundaryCondition{},
	}
	sd.Profiles[profNum].BoundaryConditions[reach] = &bcs

	// Get type and data of boundary condition
	for sc.Scan() {
		line := sc.Text()
		loe := leftofEquals(line)
		if stringInSlice(loe, steadyElementsPrefix[:]) {
			skipScan = true // a new HEC RAS element has been encountered, skip next scan and return
			return
		}

		// findout location and data of Up and Dn BCs
		switch loe {
		case "Up Type", "Dn Type":
			loc := strings.Split(loe, " ")[0]
			if entry, ok := bcs[loc]; ok {
				entry.Type = bcTypeMapping[strings.TrimSpace(rightofEquals(line))]
				bcs[loc] = entry
			}
		case "Up Known WS", "Dn Known WS":
			wse, innerErr := parseFloat(strings.TrimSpace(rightofEquals(line)), 64)
			if innerErr != nil {
				return skipScan, innerErr
			}
			loc := strings.Split(loe, " ")[0]
			if entry, ok := bcs[loc]; ok {
				entry.Data = map[string]float64{"Known WS": wse}
				bcs[loc] = entry
			}
		case "Up Slope", "Dn Slope":
			slope, innerErr := parseFloat(strings.TrimSpace(rightofEquals(line)), 64)
			if innerErr != nil {
				return skipScan, innerErr
			}
			loc := strings.Split(loe, " ")[0]
			if entry, ok := bcs[loc]; ok {
				entry.Data = map[string]float64{"Slope": slope}
				bcs[loc] = entry
			}
		case "Up Rating Curve # Pts", "Dn Rating Curve # Pts":
			pairs, innerErr := getDataPairsfromTextBlock(loe, sc, 80, 8)
			if innerErr != nil {
				return skipScan, innerErr
			}
			loc := strings.Split(loe, " ")[0]
			if entry, ok := bcs[loc]; ok {
				entry.Data = map[string][][2]float64{"Rating Curve": pairs}
				bcs[loc] = entry
			}
		}
	}
	return
}

// Get Forcing Data from steady flow file.
func getSteadyData(fd *ForcingData, fs filestore.FileStore, flowFilePath string) error {
	flowFileName := filepath.Base(flowFilePath)

	file, err := fs.GetObject(flowFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	numProf, names, err := getNameNumProfiles(&file)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	sd := SteadyData{
		Profiles: make([]Profile, numProf),
	}
	for index, element := range names {
		sd.Profiles[index].Name = element
	}

	sc := bufio.NewScanner(file)
	eof := !sc.Scan()
	if err := sc.Err(); err != nil {
		return err
	}
	skipScan := false

	sd.FlowTitle, sd.ProgramVersion, skipScan = getFlowTitleVersion(sc, steadyElementsPrefix[:])

	for !eof {
		line := sc.Text()

		switch {
		case strings.HasPrefix(line, "River Rch & RM="):
			err = getReachFlows(sc, &sd)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		case strings.HasPrefix(line, "Boundary for River Rch & Prof#="):
			skipScan, err = getReachBCs(sc, &sd)
		}

		// if a new RAS element is encountered during the functions call, scanning again will skip that element, therefore skip scan
		if !skipScan {
			eof = !sc.Scan()
			if err := sc.Err(); err != nil {
				return err
			}
		}
	}
	fd.Steady[flowFileName] = sd
	return nil
}
