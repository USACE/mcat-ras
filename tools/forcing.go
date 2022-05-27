package tools

import (
	"time"
)

// Forcing Data ...
type ForcingData struct {
	Steady        map[string][]Profile `json:"Steady,omitempty"`
	QuasiUnsteady interface{}          `json:"QuasiUnsteady,omitempty"` // to be implemented
	Unsteady      map[string][]Profile `json:"Unsteady,omitempty"`
}

// Unsteady Data
type Unsteady struct {
	InitialConditions  interface{} // to be implemented
	BoundaryConditions interface{}
	MeterologicalData  interface{} // to be implemented
	ObservedData       interface{} // to be implemented // added in version 6.2
}

// Steady Flow Profile ...
type Profile struct {
	Name                  string
	BoundaryConditions    map[string]map[string]BoundaryCondition
	Flows                 map[string][]RiverFlow
	StorageAreaElevations []StoAreaElevation
}

// Boundary Condition ...
type BoundaryCondition struct {
	RS     string      `json:"river_station,omitempty"` // only exists for unsteady rivers
	BCLine string      `json:"bc_line,omitempty"`       // only exists for unsteady storage and 2D areas
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
}

// River Flow Data Pair...
type RiverFlow struct {
	RS   float32 `json:"river_station"`
	Flow float32 `json:"flow"`
}

// Storage Area Elevation Data Pair...
type StoAreaElevation struct {
	SorageArea string  `json:"storage_area"`
	Elevation  float32 `json:"elevation"`
}

// Hydrograph Data.
// Can be Flow, Stage, Precipitation, or Gate Opening Hydrograph.
type Hydrograph struct {
	TimeInterval       int       `json:"time_interval"` // seconds
	Values             []float32 `json:"values"`
	UseFixedStart      bool      `json:"fixed_start"`
	FixedStartDateTime time.Time `json:"fixed_start_date_time,omitempty"` // HEC RAS time does not have time zone, using UTC
}

// Rating Curve Data Pair
type RatingCurveDataPair struct {
	Stage     float32 `json:"stage"`
	Elevation float32 `json:"elevation"`
}

// Elevation Controlled Gates Data.
type ElevControlGates struct {
}
