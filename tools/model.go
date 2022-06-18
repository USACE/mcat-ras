package tools

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/USACE/filestore"
	"github.com/dewberry/gdal"
	"github.com/go-errors/errors" // warning: replaces standard errors
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

var RasRE fileExtMatchers = fileExtMatchers{ // Maybe these ones are better? need a regex experts opinion
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
	Projection:  regexp.MustCompile(".pr[oj]"),
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
	Type               string
	Version            string
	DefinitionFile     string
	DefinitionFileHash string
	Files              ModelFiles
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
	Type           string
	Version        string
	FileStore      filestore.FileStore
	ModelDirectory string
	FileList       []string
	Metadata       ProjectMetadata
}

// IsAModel ...
func (rm *RasModel) IsAModel() bool {
	if len(rm.Metadata.GeomFiles) == 0 {
		fmt.Println("no geometry files identified")
		return false
	}
	return true
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
func (rm *RasModel) Index() Model {
	mod := Model{
		Type:               rm.Type,
		Version:            rm.Version,
		DefinitionFile:     rm.Metadata.ProjFilePath,
		DefinitionFileHash: rm.Metadata.ProjFileContents.Hash,
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
					Georeference:       rm.Metadata.Projection,
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
		file := filepath.Base(p.Path)
		mod.Files.InputFiles.ControlFiles.Paths = append(mod.Files.InputFiles.ControlFiles.Paths, p.Path)
		mod.Files.InputFiles.ControlFiles.Data[file] = p
	}
	for _, g := range rm.Metadata.GeomFiles {
		file := filepath.Base(g.Path)
		mod.Files.InputFiles.GeometryFiles.Paths = append(mod.Files.InputFiles.GeometryFiles.Paths, g.Path)
		mod.Files.InputFiles.GeometryFiles.FeaturesProperties[file] = g
	}
	for _, f := range rm.Metadata.FlowFiles {
		file := filepath.Base(f.Path)
		mod.Files.InputFiles.ForcingFiles.Paths = append(mod.Files.InputFiles.ForcingFiles.Paths, f.Path)
		mod.Files.InputFiles.ForcingFiles.Data[file] = f
	}

	// Need to add output and SupplementalFiles files...
	return mod
}

// Checks if a model is geospatial or not.
// Versions less than 4.0 are not considered geospatial
func (rm *RasModel) IsGeospatial() bool {
	if rm.Metadata.Projection == "" {
		fmt.Println(rm.Metadata.ProjFilePath, "| no valid coordinate reference system")
		return false
	}
	modelVersions := strings.Split(rm.Version, ",")
	for _, version := range modelVersions {
		if strings.Contains(version, ".g") {
			geomVersion := strings.TrimSpace(strings.Split(version, ":")[1])
			v, err := parseFloat(geomVersion, 64)
			if err != nil {
				fmt.Println("could not convert the geometry version to a float")
				return false
			}
			if v < 4 {
				fmt.Printf("geometry file version: %f is not geospatial", v)
				return false
			}
		}
	}

	return true
}

// GeospatialData ...
func (rm *RasModel) GeospatialData(destinationCRS int) (GeoData, error) {
	gd := GeoData{}
	if rm.IsGeospatial() {
		modelUnits := rm.Metadata.ProjFileContents.Units

		sourceCRS := rm.Metadata.Projection

		if err := checkUnitConsistency(modelUnits, sourceCRS); err != nil {
			return gd, errors.Wrap(err, 0)
		}

		gd.Features = make(map[string]Features)
		gd.Georeference = destinationCRS

		for _, g := range rm.Metadata.GeomFiles {
			if err := GetGeospatialData(&gd, rm.FileStore, g.Path, sourceCRS, destinationCRS); err != nil {
				return gd, errors.Wrap(err, 0)
			}
		}
		return gd, nil
	}
	err := errors.New("the model is not geospatial")
	return gd, errors.Wrap(err, 0)

}

func getModelFiles(rm *RasModel) error {
	prefix := filepath.Dir(rm.Metadata.ProjFilePath) + "/"

	files, err := rm.FileStore.GetDir(prefix, false)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for _, file := range *files {
		// get only files that share the same base name or .prj files for projection
		// rational behind .prj file is that there can be a shp file in the same level of Hec-RAS
		// providing potential projection
		if strings.HasPrefix(filepath.Join(file.Path, file.Name), strings.TrimSuffix(rm.Metadata.ProjFilePath, "prj")) || filepath.Ext(file.Name) == ".prj" {
			rm.FileList = append(rm.FileList, filepath.Join(file.Path, file.Name))
		}
	}

	return nil
}

// getProjection Reads a projection file. returns none to allow concurrency
func getProjection(rm *RasModel, fn string, wg *sync.WaitGroup) {

	defer wg.Done()

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Scan()
	line := sc.Text()

	sourceSpRef := gdal.CreateSpatialReference(line)
	if err := sourceSpRef.Validate(); err != nil {
		fmt.Println(fmt.Sprintf("%s is not a valid projection file.", fn))
		return
	}
	if rm.Metadata.Projection != "" {
		fmt.Println("Projection already exist for the model.")
		return
	}

	rm.Metadata.Projection = line

	return
}

// NewRasModel ...
func NewRasModel(key string, fs filestore.FileStore) (*RasModel, error) {
	rm := RasModel{ModelDirectory: filepath.Dir(key), FileStore: fs, Type: "RAS"}

	err := verifyPrjPath(key, &rm)
	if err != nil {
		return &rm, errors.Wrap(err, 0)
	}

	err = getModelFiles(&rm)
	if err != nil {
		return &rm, errors.Wrap(err, 0)
	}

	err = getPrjData(&rm)
	if err != nil {
		return &rm, errors.Wrap(err, 0)
	}

	var rasWG rasWaitGroup

	// get projection using name.projection file
	rasWG.Projection.Add(1)
	projecFile := strings.TrimSuffix(key, ".prj") + ".projection"
	go getProjection(&rm, projecFile, &rasWG.Projection)

	for _, fp := range rm.FileList {

		ext := filepath.Ext(fp)

		switch {

		case RasRE.Plan.MatchString(ext):
			rasWG.Plan.Add(1)
			go getPlanData(&rm, fp, &rasWG.Plan)

		case RasRE.Geom.MatchString(ext):
			rasWG.Geom.Add(1)
			go getGeomData(&rm, fp, &rasWG.Geom)

		case RasRE.AllFlow.MatchString(ext):
			rasWG.Flow.Add(1)
			go getFlowData(&rm, fp, &rasWG.Flow)

		case rm.Metadata.Projection == "" && RasRE.Projection.MatchString(ext):
			if filepath.Base(key) != filepath.Base(fp) && fp != projecFile {
				rasWG.Projection.Add(1)
				go getProjection(&rm, fp, &rasWG.Projection)
			}

		}
	}

	rasWG.Plan.Wait()
	rasWG.Geom.Wait()
	rasWG.Flow.Wait()
	rasWG.Projection.Wait()

	for _, p := range rm.Metadata.PlanFiles {
		version := p.ProgramVersion
		if version != "" {
			rm.Version += fmt.Sprintf("%s: %s, ", p.FileExt, version)
		}
	}
	for _, g := range rm.Metadata.GeomFiles {
		version := g.ProgramVersion
		if version != "" {
			rm.Version += fmt.Sprintf("%s: %s, ", g.FileExt, version)
		}
	}
	for _, f := range rm.Metadata.FlowFiles {
		version := f.ProgramVersion
		if version != "" {
			rm.Version += fmt.Sprintf("%s: %s, ", f.FileExt, version)
		}
	}

	if len(rm.Version) >= 2 {
		rm.Version = rm.Version[0 : len(rm.Version)-2]
	}

	return &rm, nil
}
