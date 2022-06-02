package tools

import (
	"fmt"
	"path/filepath"

	"github.com/USACE/filestore"
)

// These prefixes are used to determine the beginning and end of HEC-RAS elements
var forcingElementsPrefix = [...]string{"Boundary Location"}

// Forcing Data ...
type ForcingData struct {
	Steady        map[string][]Profile    `json:"Steady,omitempty"`
	QuasiUnsteady interface{}             `json:"QuasiUnsteady,omitempty"` // to be implemented
	Unsteady      map[string]UnsteadyData `json:"Unsteady,omitempty"`
}

// Boundary Condition ...
type BoundaryCondition struct {
	RS          string      `json:",omitempty"`        // only exists for unsteady rivers
	BCLine      string      `json:"bc_line,omitempty"` // only exists for unsteady storage and 2D areas
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
}

// Hydrograph Data.
// Can be Flow, Stage, Precipitation, Uniform Lateral Inflow, Lateral Inflow, Ground Water Interflow, or Gate Opening Hydrograph.
type Hydrograph struct {
	TimeInterval       string      `json:"time_interval,omitempty"`
	EndRS              string      `json:"flow_distribution_last_RS,omitempty"` // flow will be distributed from RS to EndRS. Valid for Reaches with Uniform Lateral Inflow or Groundwater Interflow
	Values             interface{} `json:"values,omitempty"`
	UseDSS             bool        `json:"use_dss"`
	UseFixedStart      bool        `json:"fixed_start"`
	FixedStartDateTime *DateTime   `json:"fixed_start_date_time,omitempty"` // pointer to have zero value, so that omitempty can work
}

type DateTime struct {
	Date  string `json:"date,omitempty"`
	Hours string `json:"hours,omitempty"` // should not be int/float or else 0015 hours will become 15 hours
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
