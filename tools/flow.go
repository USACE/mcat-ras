package tools

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// FlowFileContents keywords  and data container for ras flow file search
type FlowFileContents struct {
	Path                string
	Hash                string
	FileExt             string //`json:"File Extension"`
	FlowTitle           string //`json:"Flow Title"`
	ProgramVersion      string //`json:"Program Version"`
	NProfiles           string //`json:"Number of Profiles"`
	ProfileNames        string //`json:"Profile Names"`
	UpdatedProfileNames string //`json:"Updated Profile Names"`
	Notes               string //`json:"Notes"`
}

// getGeomData Reads a flow file. returns none to allow concurrency
func getFlowData(rm *RasModel, fn string, wg *sync.WaitGroup) {

	defer wg.Done()

	meta := FlowFileContents{Path: fn, FileExt: filepath.Ext(fn)}

	var err error
	msg := fmt.Sprintf("%s failed to process.", filepath.Base(fn))
	defer func() {
		meta.Notes += msg
		rm.Metadata.FlowFiles = append(rm.Metadata.FlowFiles, meta)
		if err != nil {
			log.Println(err)
		}
	}()

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return
	}
	defer f.Close()

	hasher := sha256.New()

	fs := io.TeeReader(f, hasher) // fs is still a stream
	sc := bufio.NewScanner(fs)

	var line string
	for sc.Scan() {

		line = sc.Text()

		match, err := regexp.MatchString("=", line)
		if err != nil {
			return
		}

		if match {
			data := strings.Split(line, "=")

			switch data[0] {

			case "Flow Title":
				meta.FlowTitle = data[1]

			case "Number of Profiles":
				meta.NProfiles = data[1]

			case "Profile Names":
				meta.ProfileNames = data[1]

			case "Program Version":
				meta.ProgramVersion = data[1]

			}
		}
	}

	msg = ""
	meta.Hash = fmt.Sprintf("%x", hasher.Sum(nil))

	return
}
