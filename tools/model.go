package tools

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/USACE/filestore"
)

type fileExtMatchers struct {
	Geom        *regexp.Regexp
	Plan        *regexp.Regexp
	Steady      *regexp.Regexp
	Unsteady    *regexp.Regexp
	QuasiSteady *regexp.Regexp
	AllFlow     *regexp.Regexp
	Output      *regexp.Regexp
	SteadyRun   *regexp.Regexp
	UnsteadyRun *regexp.Regexp
	AllFlowRun  *regexp.Regexp
	Projection  *regexp.Regexp
}

var rasRE fileExtMatchers = fileExtMatchers{ // Maybe these ones are better? need a regex experts opinion
	Geom:        regexp.MustCompile(".g[0-9][0-9]"),     // `^\.g(0[1-9]|[1-9][0-9])$`
	Plan:        regexp.MustCompile(".p[0-9][0-9]"),     // `^\.p(0[1-9]|[1-9][0-9])$`
	Steady:      regexp.MustCompile(".f[0-9][0-9]"),     // `^\.f(0[1-9]|[1-9][0-9])$`
	Unsteady:    regexp.MustCompile(".u[0-9][0-9]"),     // `^\.u(0[1-9]|[1-9][0-9])$`
	QuasiSteady: regexp.MustCompile(".q[0-9][0-9]"),     // `^\.q(0[1-9]|[1-9][0-9])$`
	AllFlow:     regexp.MustCompile(".[fqu][0-9][0-9]"), // `^\.[fqu](0[1-9]|[1-9][0-9])$`
	Output:      regexp.MustCompile(".O[0-9][0-9]"),     // `^\.O(0[1-9]|[1-9][0-9])$`
	SteadyRun:   regexp.MustCompile(".r[0-9][0-9]"),     // `^\.r(0[1-9]|[1-9][0-9])$`
	UnsteadyRun: regexp.MustCompile(".x[0-9][0-9]"),     // `^\.x(0[1-9]|[1-9][0-9])$`
	AllFlowRun:  regexp.MustCompile(".[rx][0-9][0-9]"),  // `^\.[rx](0[1-9]|[1-9][0-9])$`
	Projection:  regexp.MustCompile(".projection"),
}

// holder of multiple wait groups to help process files concurrency
type rasWaitGroup struct {
	Geom       sync.WaitGroup
	Plan       sync.WaitGroup
	Flow       sync.WaitGroup
	Projection sync.WaitGroup
}

// Model is a general type should contain all necessary data for a model of any type.
type Model struct {
	Type           string
	Version        string
	DefinitionFile string
	Files          ModelFiles
}

// ModelFiles ...
type ModelFiles struct {
	InputFiles        InputFiles
	OutputFiles       OutputFiles
	SupplementalFiles SupplementalFiles
}

// InputFiles is a general type that should contain all data pulled from the models input files
type InputFiles struct {
	ControlFiles        ControlFiles
	ForcingFiles        ForcingFiles
	GeometryFiles       GeometryFiles
	SimulationVariables interface{} // placeholder
	LocalVariables      interface{} // placeholder
}

// ControlFiles ...
type ControlFiles struct {
	Paths []string
	Data  map[string]interface{} // placeholder
}

// ForcingFiles ...
type ForcingFiles struct {
	Paths []string
	Data  map[string]interface{} // placeholder
}

// GeometryFiles is a general type that should contain all data pulled from the models spatial files
type GeometryFiles struct {
	Paths              []string
	FeaturesProperties map[string]interface{} // placeholder
	Georeference       interface{}            // placeholder
}

// OutputFiles is a general type that should contain all data pulled from the models output files
type OutputFiles struct {
	Paths           []string
	ModelPrediction interface{} // placeholder
	RunFiles        []string
	RunLogs         []string
}

// SupplementalFiles is a general type that should contain all data pulled from the models supplemental files
type SupplementalFiles struct {
	Paths             []string
	Visulizations     interface{} // placeholder
	ObservationalData interface{} // placeholder
}

// RasModel ...
type RasModel struct {
	FileStore      filestore.FileStore
	ModelDirectory string
	Version        string
	Type           string
	Metadata       ProjectMetadata
	isModel        bool
	FileList       []string
}

// IsAModel ...
func (rm *RasModel) IsAModel() bool {
	return rm.isModel
}

// IsGeospatial ...
func (rm *RasModel) IsGeospatial() bool {
	if rm.Metadata.GeomFiles[0].FileExt != "" {
		return true
	}
	return false
}

// ModelType ...
func (rm *RasModel) ModelType() string {
	return rm.Type
}

// ModelVersion ...
func (rm *RasModel) ModelVersion() string {
	return rm.Version
}

// Index ...
func (rm *RasModel) Index() (Model, error) {
	if !rm.IsAModel() {
		return Model{}, errors.New("model is not valid")
	}

	mod := Model{
		Type:           rm.Type,
		Version:        rm.Version,
		DefinitionFile: rm.Metadata.ProjFilePath,
		Files: ModelFiles{
			InputFiles: InputFiles{
				ControlFiles: ControlFiles{
					Paths: make([]string, 0),
					Data:  make(map[string]interface{}),
				},
				ForcingFiles: ForcingFiles{
					Paths: make([]string, 0),
					Data:  make(map[string]interface{}),
				},
				GeometryFiles: GeometryFiles{
					Paths:              make([]string, 0),
					FeaturesProperties: make(map[string]interface{}),
					Georeference:       nil,
				},
				SimulationVariables: nil,
				LocalVariables:      nil,
			},
			OutputFiles: OutputFiles{
				Paths:           make([]string, 0),
				ModelPrediction: nil,
				RunFiles:        make([]string, 0),
				RunLogs:         make([]string, 0),
			},
			SupplementalFiles: SupplementalFiles{
				Paths:             make([]string, 0),
				Visulizations:     nil,
				ObservationalData: nil,
			},
		},
	}

	for _, p := range rm.Metadata.PlanFiles {
		mod.Files.InputFiles.ControlFiles.Paths = append(mod.Files.InputFiles.ControlFiles.Paths, p.Path)
		mod.Files.InputFiles.ControlFiles.Data["PlanTitle"] = p.PlanTitle
	}
	for _, g := range rm.Metadata.GeomFiles {
		mod.Files.InputFiles.GeometryFiles.Paths = append(mod.Files.InputFiles.GeometryFiles.Paths, g.Path)
		mod.Files.InputFiles.GeometryFiles.FeaturesProperties["GeomTitle"] = g.GeomTitle
	}
	for _, f := range rm.Metadata.FlowFiles {
		mod.Files.InputFiles.ForcingFiles.Paths = append(mod.Files.InputFiles.ForcingFiles.Paths, f.Path)
		mod.Files.InputFiles.ForcingFiles.Data["FlowTitle"] = f.FlowTitle
	}

	// need to add output files and supplemental files...

	return mod, nil
}

// GeospatialData ...
func (rm *RasModel) GeospatialData(destinationCRS int) (GeoData, error) {
	gd := GeoData{Features: make(map[string]Features), Georeference: destinationCRS}

	sourceCRS := rm.Metadata.Projection
	for _, g := range rm.Metadata.GeomFiles {
		if err := GetGeospatialData(&gd, rm.FileStore, g.Path, sourceCRS, destinationCRS); err != nil {
			return gd, err
		}
	}
	return gd, nil
}

func getModelFiles(rm *RasModel) error {
	prefix := filepath.Dir(rm.Metadata.ProjFilePath) + "/"

	files, err := rm.FileStore.GetDir(prefix, false)
	if err != nil {
		return err
	}

	for _, file := range *files {
		if filepath.Ext(file.Name) != ".prj" {
			rm.FileList = append(rm.FileList, filepath.Join(file.Path, file.Name))
		}
	}

	return nil
}

// getProjection Reads a projection file. returns none to allow concurrency
func getProjection(rm *RasModel, fn string, wg *sync.WaitGroup, errChan chan error) {

	defer wg.Done()

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		errChan <- err
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Scan()
	line := sc.Text()
	if strings.HasPrefix(line, "PROJCS[") {
		rm.Metadata.Projection = line
		return
	}

	errChan <- errors.New("Unexpected projection file structure")
	return
}

// NewRasModel ...
func NewRasModel(key string, fs filestore.FileStore) (*RasModel, error) {
	rm := RasModel{ModelDirectory: filepath.Dir(key), FileStore: fs, Type: "RAS"}

	err := verifyPrjPath(key, &rm)
	if err != nil {
		return &rm, err
	}

	err = getModelFiles(&rm)
	if err != nil {
		return &rm, err
	}

	err = getPrjData(&rm)
	if err != nil {
		return &rm, err
	}

	errChan := make(chan error)
	var rasWG rasWaitGroup

	nProjection := 0
	for _, fp := range rm.FileList {

		ext := filepath.Ext(fp)

		switch {

		case rasRE.Plan.MatchString(ext):
			rasWG.Plan.Add(1)
			go getPlanData(&rm, fp, &rasWG.Plan, errChan)

		case rasRE.Geom.MatchString(ext):
			rasWG.Geom.Add(1)
			go getGeomData(&rm, fp, &rasWG.Geom, errChan)

		case rasRE.AllFlow.MatchString(ext):
			rasWG.Flow.Add(1)
			go getFlowData(&rm, fp, &rasWG.Flow, errChan)

		case rasRE.Projection.MatchString(ext):
			if nProjection > 0 {
				return &rm, errors.New("multiple projection files identified")
			}
			rasWG.Projection.Add(1)
			go getProjection(&rm, fp, &rasWG.Projection, errChan)
			nProjection++

		}
	}

	rasWG.Plan.Wait()
	rasWG.Geom.Wait()
	rasWG.Flow.Wait()
	rasWG.Projection.Wait()

	if len(errChan) > 0 {
		fmt.Printf("Encountered %d errors\n", len(errChan))
		return &rm, <-errChan
	}

	for _, p := range rm.Metadata.PlanFiles {
		rm.Version += fmt.Sprintf("%s: %s, ", p.FileExt, p.ProgramVersion)
	}
	for _, g := range rm.Metadata.GeomFiles {
		rm.Version += fmt.Sprintf("%s: %s, ", g.FileExt, g.ProgramVersion)
	}
	for _, f := range rm.Metadata.FlowFiles {
		rm.Version += fmt.Sprintf("%s: %s, ", f.FileExt, f.ProgramVersion)
	}

	if len(rm.Version) >= 2 {
		rm.Version = rm.Version[0 : len(rm.Version)-2]
	}
	rm.isModel = true

	return &rm, nil
}
