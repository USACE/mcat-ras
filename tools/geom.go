package tools

import (
	"bufio"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

// GeomFileContents keywords  and data container for ras flow file search
type GeomFileContents struct {
	Path           string
	FileExt        string                 `json:"File Extension"`
	GeomTitle      string                 `json:"Geom Title"`
	ProgramVersion string                 `json:"Program Version"`
	Description    string                 `json:"Description"`
	Structures     []hydraulicStructures  `json:"Hydraulic Structures"`
	StorageAreas   map[string]storageArea `json:"Storage Areas"`
	TwoDAreas      map[string]twoDArea    `json:"2D Areas"`
	Connections    map[string]connection  `json:"Connections"`
	Notes          string
}

// getGeomData Reads a geometry file. returns none to allow concurrency
func getGeomData(rm *RasModel, fn string, wg *sync.WaitGroup) {

	defer wg.Done()

	meta := GeomFileContents{
		Path:         fn,
		FileExt:      filepath.Ext(fn),
		StorageAreas: make(map[string]storageArea),
		TwoDAreas:    make(map[string]twoDArea),
		Connections:  make(map[string]connection),
	}

	var err error
	msg := fmt.Sprintf("%s failed to process.", filepath.Base(fn))
	defer func() {
		meta.Notes += msg
		rm.Metadata.GeomFiles = append(rm.Metadata.GeomFiles, meta)
		if err != nil {
			log.Println(err)
		}
	}()

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	var description string

	header := true
	idx := 0
	for sc.Scan() {
		idx++
		line := sc.Text()
		switch {

		case strings.HasPrefix(line, "Geom Title="):
			meta.GeomTitle = rightofEquals(line)

		case strings.HasPrefix(line, "Program Version="):
			meta.ProgramVersion = rightofEquals(line)

		case strings.HasPrefix(line, "BEGIN GEOM DESCRIPTION:"):
			if header {
				description, idx, err = getDescription(sc, idx, "END GEOM DESCRIPTION:")
				if err != nil {
					return
				}
				meta.Description += description
			}

		case strings.HasPrefix(line, "River Reach="):
			structures, err := getHydraulicStructureData(rm, fn, idx)
			if err != nil {
				log.Println("Hydraulic Structures|", meta.FileExt, err)
				continue
			}
			meta.Structures = append(meta.Structures, structures)
			header = false

		case strings.HasPrefix(line, "Storage Area="):
			areaName, areaData, err := getAreasData(rm, fn, idx)
			if err != nil {
				log.Println("SA/2D Areas|", meta.FileExt, err)
				continue
			}
			switch areaData.(type) {

			case storageArea:
				meta.StorageAreas[areaName] = areaData.(storageArea)

			case twoDArea:
				meta.TwoDAreas[areaName] = areaData.(twoDArea)
			}
			header = false

		case strings.HasPrefix(line, "Connection="):
			connName, connecData, err := getConnectionsData(rm, fn, idx)
			if err != nil {
				log.Println("Connections|", meta.FileExt, err)
				continue
			}
			meta.Connections[connName] = connecData
			header = false

		case strings.HasPrefix(line, "BC Line Name="):
			bcArea, bc, err := getBCLineData(rm, fn, idx)
			if err != nil {
				log.Println("BC Line |", meta.FileExt, err)
				continue
			}
			if val, ok := meta.StorageAreas[bcArea]; ok {
				val.NumBCLines++
				val.BCLines = append(val.BCLines, bc)
				meta.StorageAreas[bcArea] = val
			} else if val, ok := meta.TwoDAreas[bcArea]; ok {
				val.NumBCLines++
				val.BCLines = append(val.BCLines, bc)
				meta.TwoDAreas[bcArea] = val
			}
			header = false

		}
	}
	msg = ""
	return
}
