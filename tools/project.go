package tools

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/USACE/filestore"
	"github.com/go-errors/errors" // warning: replaces standard errors
)

// ProjectMetadata contains information scraped from all files listed in the .prj file
type ProjectMetadata struct {
	ProjFilePath     string
	ProjFileContents PrjFileContents    //`json:"Project Data"`
	PlanFiles        []PlanFileContents //`json:"Plan Data"`
	FlowFiles        []FlowFileContents //`json:"Flow Data"`
	GeomFiles        []GeomFileContents //`json:"Geometry Data"`
	Projection       string             //`json:"Projection"`
	Notes            string             //`json:"Notes"`
}

// PrjFileContents keywords  and data container for ras project file search
type PrjFileContents struct {
	ProjTitle       string   //`json:"Proj Title"`
	PlanFile        []string //`json:"Plan File"`
	FlowFile        []string //`json:"Flow File"`
	QuasiSteadyFile []string //`json:"QuasiSteady File"`
	UnsteadyFile    []string //`json:"Unsteady File"`
	GeomFile        []string //`json:"Geom File"`
	Units           string   //`json:"Units"`
	CurrentPlan     string   //`json:"Current Plan"`
	Description     string   //`json:"Description"`
} //

func readFirstLine(fs filestore.FileStore, fn string) (string, error) {
	file, err := fs.GetObject(fn)
	if err != nil {
		fmt.Println("Couldnt open the file", fn)
		return "", errors.Wrap(err, 0)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	return rmNewLineChar(line), errors.Wrap(err, 0)
}

func rmNewLineChar(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}

// verifyPrjPath identifies the .prj file within the passed model directory ...
func verifyPrjPath(key string, rm *RasModel) error {

	if filepath.Ext(key) != ".prj" {
		return errors.Errorf("%s is not a .prj file", key)
	}

	firstLine, err := readFirstLine(rm.FileStore, key)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	if !strings.Contains(firstLine, "Proj Title=") {
		return errors.Errorf("%s is not a RAS Project file", key)
	}

	rm.Metadata.ProjFilePath = key

	return nil
}

// getPrjData reads a Project file and returns data of interest
func getPrjData(rm *RasModel) error {

	meta := PrjFileContents{}

	f, err := rm.FileStore.GetObject(rm.Metadata.ProjFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var line string
	for sc.Scan() {
		line = sc.Text()

		match, err := regexp.MatchString("=", line)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		beginDescription, err := regexp.MatchString("BEGIN DESCRIPTION", line)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		units, err := regexp.MatchString("Units", line)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if match {
			data := strings.Split(line, "=")

			switch data[0] {

			case "Proj Title":
				meta.ProjTitle = data[1]

			case "Plan File":
				meta.PlanFile = append(meta.PlanFile, data[1])

			case "Flow File":
				meta.FlowFile = append(meta.FlowFile, data[1])

			case "QuasiSteady File":
				meta.QuasiSteadyFile = append(meta.QuasiSteadyFile, data[1]) //Does this exist?

			case "Unsteady File":
				meta.UnsteadyFile = append(meta.UnsteadyFile, data[1]) //Does this exist?

			case "Geom File":
				meta.GeomFile = append(meta.GeomFile, data[1])

			case "Current Plan":
				meta.CurrentPlan = data[1]

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

		} else if units {
			meta.Units = line
		}
	}

	rm.Metadata.ProjFileContents = meta
	return nil
}
