package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Dewberry/mcat-ras/config"
	"github.com/Dewberry/mcat-ras/tools"

	"github.com/USACE/filestore"
	"github.com/dewberry/gdal"
	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// GeospatialData godoc
// @Summary Extract geospatial data
// @Description Extract geospatial data from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} interface{}
// @Failure 500 {object} SimpleResponse
// @Router /geospatialdata [get]
func GeospatialData(ac *config.APIConfig) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		if !isAModel(ac.FileStore, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		if !isGeospatial(definitionFile, *ac.FileStore) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not geospatial.")
		}

		data, err := geospatialData(definitionFile, ac.FileStore, ac.DestinationCRS)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, fmt.Sprintf("Go error encountered: %v", err.Error()), err.(*errors.Error).ErrorStack()})
		}

		return c.JSON(http.StatusOK, data)
	}
}

func geospatialData(definitionFile string, fs *filestore.FileStore, destinationCRS int) (tools.GeoData, error) {
	gd := tools.GeoData{Features: make(map[string]tools.Features), Georeference: destinationCRS}

	mfiles, err := modFiles(definitionFile, *fs)
	if err != nil {
		return gd, errors.Wrap(err, 0)
	}

	projecFile := strings.TrimSuffix(definitionFile, ".prj") + ".projection"
	proj, err := getProjection(*fs, projecFile)
	if err != nil {
		return gd, errors.Wrap(err, 0)
	}

	for _, fp := range mfiles {

		ext := filepath.Ext(fp)

		switch {

		case tools.RasRE.Geom.MatchString(ext):

			if err := tools.GetGeospatialData(&gd, *fs, fp, proj, destinationCRS); err != nil {
				return gd, errors.Wrap(err, 0)
			}

		}
	}

	return gd, nil
}

func getProjection(fs filestore.FileStore, fn string) (string, error) {

	f, err := fs.GetObject(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Scan()
	line := sc.Text()

	sourceSpRef := gdal.CreateSpatialReference(line)

	return line, sourceSpRef.Validate()
}
