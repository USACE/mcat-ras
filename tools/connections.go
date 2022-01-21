package tools

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/go-errors/errors" // warning: replaces standard errors
)

// Store HEC-RAS SA/2D Area Connections
type connection struct {
	Description string      `json:"Description"`
	UpSA        string      `json:"Up SA"`
	DnSA        string      `json:"Dn SA"`
	WeirWidth   float64     `json:"Weir Width"`
	WeirElev    maxMinPairs `json:"Weir Elevations"`
	NumGates    int         `json:"Num Gates"`
	Gates       []gates
	NumConduits int        `json:"Num Culvert Conduits"`
	Conduits    []conduits `json:"Culvert Conduits"`
}

// Extract data from Connections
func getConnectionsData(rm *RasModel, fn string, i int) (string, connection, error) {
	name := ""
	connection := connection{}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return name, connection, errors.Wrap(err, 0) 
	}
	defer f.Close()

	cSc := bufio.NewScanner(f)

	ci := 0
	for cSc.Scan() {
		ci++
		if ci == i {
			lineData := strings.Split(rightofEquals(cSc.Text()), ",")
			name = strings.TrimSpace(lineData[0])
		} else if ci > i {
			line := cSc.Text()
			switch {
				
			case strings.HasPrefix(line, "Connection Desc="):
				description, err := getDescriptionConnections(cSc, "Connection Line=")
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				connection.Description += description

			case strings.HasPrefix(line, "Connection Up SA="):
				connection.UpSA = rightofEquals(line)

			case strings.HasPrefix(line, "Connection Dn SA="):
				connection.DnSA = rightofEquals(line)

			case strings.HasPrefix(line, "Conn Weir WD="):
				weirWidth, err := strconv.ParseFloat(rightofEquals(line), 64)
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				connection.WeirWidth = weirWidth

			case strings.HasPrefix(line, "Conn Weir SE="):
				nElev, err := strconv.Atoi(rightofEquals(line))
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				nLines := numberofLines(nElev*2, 80, 8)

				elev, _, err := getMaxMinElev(cSc, 0, nLines, 0, 80, 8, 2)
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				connection.WeirElev = elev

			case strings.HasPrefix(line, "Conn Gate Name Wd,H,"):
				cSc.Scan()
				gate, err := getGates(cSc.Text())
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				connection.Gates = append(connection.Gates, gate)
				connection.NumGates++

			case strings.HasPrefix(line, "Connection Culv="):
				conduit, err := getConduits(line, false)
				if err != nil {
					return name, connection, errors.Wrap(err, 0) 
				}
				connection.Conduits = append(connection.Conduits, conduit)
				connection.NumConduits++

			case strings.HasPrefix(line, "Conn Outlet Rating Curve="):
				return name, connection, nil
				
			case strings.HasPrefix(line, "Connection="):
				// guard to make sure new Connection don't overwrite previous values
				// return with whatever data is available
				return name, connection, nil
			}
		}
	}
	return name, connection, nil
}