package tools

// Steady Flow Profile ...
type Profile struct {
	Name                  string
	BoundaryConditions    map[string]map[string]BoundaryCondition
	Flows                 map[string][]RiverFlow
	StorageAreaElevations []StoAreaElevation
}

// River Flow Data Pair...
type RiverFlow struct {
	RS   float64 `json:"river_station"`
	Flow float64 `json:"flow"`
}

// Storage Area Elevation Data Pair...
type StoAreaElevation struct {
	SorageArea string  `json:"storage_area"`
	Elevation  float64 `json:"elevation"`
}
