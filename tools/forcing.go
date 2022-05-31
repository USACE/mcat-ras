package tools

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/USACE/filestore"
)

// These prefixes are used to determine the beginning and end of HEC-RAS elements
var forcingElementsPrefix = [...]string{"Boundary Condition"}

// Forcing Data ...
type ForcingData struct {
	Steady        map[string][]Profile    `json:"Steady,omitempty"`
	QuasiUnsteady interface{}             `json:"QuasiUnsteady,omitempty"` // to be implemented
	Unsteady      map[string]UnsteadyData `json:"Unsteady,omitempty"`
}

// Boundary Condition ...
type BoundaryCondition struct {
	RS          string      `json:"river_station,omitempty"` // only exists for unsteady rivers
	BCLine      string      `json:"bc_line,omitempty"`       // only exists for unsteady storage and 2D areas
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
}

// Hydrograph Data.
// Can be Flow, Stage, Precipitation, or Gate Opening Hydrograph.
type Hydrograph struct {
	TimeInterval       int       `json:"time_interval"` // seconds
	Values             []float64 `json:"values"`
	UseFixedStart      bool      `json:"fixed_start"`
	FixedStartDateTime time.Time `json:"fixed_start_date_time,omitempty"` // HEC RAS time does not have time zone, using UTC
}

// Rating Curve Data Pair
type RatingCurveDataPair struct {
	Stage     float64 `json:"stage"`
	Elevation float64 `json:"elevation"`
}

// Elevation Controlled Gates Data.
type ElevControlGates struct {
}

// Get Forcing Data from steady, unsteady or quasi-steady flow file.
func GetForcingData(fd *ForcingData, fs filestore.FileStore, flowFilePath string) (err error) {
	extPrefix := filepath.Ext(flowFilePath)[0:2]

	if extPrefix == ".f" {
		fmt.Sprintf("found steady flow file %s", flowFilePath)
		// err = getSteadyData(fd, fs, flowFilePath)
	} else if extPrefix == ".u" {
		err = getUnsteadyData(fd, fs, flowFilePath)
	} else if extPrefix == ".q" {
		return err // not implemented
	}

	return err
}