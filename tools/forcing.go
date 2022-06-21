// Structs and functions used to parse all [steady, unsteady, quasi-unsteady] types of flow files.

package tools

import (
	"path/filepath"
	"sync"

	"github.com/USACE/filestore"
)

// Main struct for focing data.
type ForcingData struct {
	Steady        map[string]SteadyData   `json:"Steady,omitempty"`
	QuasiUnsteady map[string]interface{}  `json:"QuasiUnsteady,omitempty"` // to be implemented
	Unsteady      map[string]UnsteadyData `json:"Unsteady,omitempty"`
}

// Boundary Condition.
type BoundaryCondition struct {
	RS          string      `json:",omitempty"`            // only exists for unsteady rivers
	BCLine      string      `json:"bc_line,omitempty"`     // only exists for unsteady storage and 2D areas
	Description string      `json:"description,omitempty"` // only exists for Rules, not implemented yet
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
}

// Get Forcing Data from steady, unsteady or quasi-steady flow file.
func GetForcingData(fd *ForcingData, fs filestore.FileStore, flowFilePath string, c chan error, mu *sync.Mutex) {
	extPrefix := filepath.Ext(flowFilePath)[0:2]
	var err error

	if extPrefix == ".f" {
		err = getSteadyData(fd, fs, flowFilePath, mu)
	} else if extPrefix == ".u" {
		err = getUnsteadyData(fd, fs, flowFilePath, mu)
	} else if extPrefix == ".q" {
		flowFileName := filepath.Base(flowFilePath)
		fd.QuasiUnsteady[flowFileName] = "Not Implemented"
	}

	c <- err
}
