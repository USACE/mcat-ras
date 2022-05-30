package tools

// Unsteady Data
type Unsteady struct {
	InitialConditions  interface{} // to be implemented
	BoundaryConditions interface{}
	MeterologicalData  interface{} // to be implemented
	ObservedData       interface{} // to be implemented // added in version 6.2
}
