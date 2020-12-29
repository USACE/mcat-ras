package tools

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// PlanFileContents keywords and data container for ras plan file search
type PlanFileContents struct {
	Path            string
	FileExt         string //`json:"File Extension"`
	PlanTitle       string //`json:"Plan Title"`
	ShortIdentifier string //`json:"Short Identifier"`
	ProgramVersion  string //`json:"Program Version"`
	QuasiSteadyFile string //`json:"QuasiSteady File"` //This is not currently used
	UnsteadyFile    string //`json:"Unsteady File"`    //This is not currently used
	GeomFile        string //`json:"Geom File"`
	FlowFile        string //`json:"Flow File"`
	FlowRegime      string //`json:"FlowRegime"`
	Description     string //`json:"Description"`
}

// getPlanData Reads a plan file. returns none to allow concurrency
func getPlanData(rm *RasModel, fn string, wg *sync.WaitGroup, errChan chan error) {

	defer wg.Done()

	meta := PlanFileContents{Path: fn, FileExt: filepath.Ext(fn)}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		errChan <- err
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var line string
	for sc.Scan() {

		line = sc.Text()

		match, err := regexp.MatchString("=", line)

		if err != nil {
			errChan <- err
			return
		}

		beginDescription, err := regexp.MatchString("BEGIN DESCRIPTION", line)

		if err != nil {
			errChan <- err
			return
		}

		flowRegime, err := regexp.MatchString("Subcritical|Supercritical|Mixed", line)

		if err != nil {
			errChan <- err
			return
		}

		if match {
			data := strings.Split(line, "=")

			switch data[0] {

			case "Plan Title":
				meta.PlanTitle = data[1]

			case "Short Identifier":
				meta.ShortIdentifier = data[1]

			case "Program Version":
				meta.ProgramVersion = data[1]

			case "Geom File":
				meta.GeomFile = data[1]

			case "Flow File":
				meta.FlowFile = data[1]

			}

		} else if beginDescription {

			for sc.Scan() {
				line = sc.Text()
				endDescription, _ := regexp.MatchString("END DESCRIPTION", line)

				if endDescription {
					break

				} else {
					if line != "" {
						meta.Description += line + "\n"
					}
				}

			}

		} else if flowRegime {
			meta.FlowRegime = line
		}
	}

	rm.Metadata.PlanFiles = append(rm.Metadata.PlanFiles, meta)
	return

}
