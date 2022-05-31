package tools

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/USACE/filestore"
	"github.com/go-errors/errors"
)

// Unsteady Data
type UnsteadyData struct {
	InitialConditions  interface{} // to be implemented
	BoundaryConditions UnsteadyBoundaryConditions
	MeterologicalData  interface{} // to be implemented
	ObservedData       interface{} // to be implemented // added in version 6.2
}

// Unsteady Boundary Conditions
type UnsteadyBoundaryConditions struct {
	Reaches      map[string][]BoundaryCondition
	Areas        map[string][]BoundaryCondition
	Connections  map[string][]BoundaryCondition
	PumpStations map[string]BoundaryCondition
}

// Get Boundary Condition's parent
func getBoundaryCondition(sc *bufio.Scanner) (string, string, BoundaryCondition, error) {
	parentType := "" // either Reaches, Connections, Areas, or Pump Stations
	parent := ""     // e.g. name of the river - reach, or name of Storage Area
	bc := BoundaryCondition{}

	bcArray := strings.Split(rightofEquals(sc.Text()), ",")
	if strings.TrimSpace(bcArray[0]) != "" {
		parent = fmt.Sprintf("%s - %s", strings.TrimSpace(bcArray[0]), strings.TrimSpace(bcArray[1]))
		parentType = "Reach"
		bc.RS = strings.TrimSpace(bcArray[2])
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
		return parentType, parent, bc, errors.New("Cannot determine if Boundary Condition is for a Reach, Connection, Area, or Pump Station.")
	}

	return parentType, parent, bc, nil
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

	for sc.Scan() {
		line := sc.Text()

		switch {
		case strings.HasPrefix(line, "Boundary Location="):
			parentType, parent, bc, err := getBoundaryCondition(sc)
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
	}
	fd.Unsteady[flowFileName] = ud
	return nil
}
