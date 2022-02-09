package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/USACE/filestore" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// IsGeospatial godoc
// @Summary Check if the RAS model has geospatial information
// @Description Check if the RAS model has geospatial information
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} bool
// @Router /isgeospatial [get]
func IsGeospatial(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		if !isAModel(fs, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		return c.JSON(http.StatusOK, isGeospatial(definitionFile, *fs))
	}
}

func isGeospatial(definitionFile string, fs filestore.FileStore) bool {

	modelVersions, err := getVersions(definitionFile, fs)
	if err != nil {
		return false
	}

	for _, version := range strings.Split(modelVersions, ",") {
		if strings.Contains(version, ".g") {
			geomVersion := strings.TrimSpace(strings.Split(version, ":")[1])
			v, err := strconv.ParseFloat(geomVersion, 64)
			if err != nil {
				fmt.Printf("could not convert the geometry version to a float. prj file: %s\n", definitionFile)
				return false
			}
			if v < 4 {
				fmt.Printf("geometry file version: %f is not geospatial\n", v)
				return false
			}
		}
	}

	return true
}
