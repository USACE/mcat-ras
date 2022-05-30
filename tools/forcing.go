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

// Boundary Condition ...
type BoundaryCondition struct {
	RS     string      `json:"river_station,omitempty"` // only exists for unsteady rivers
	BCLine string      `json:"bc_line,omitempty"`       // only exists for unsteady storage and 2D areas
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
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
