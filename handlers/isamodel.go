package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"app/tools"

	"github.com/USACE/filestore"
	"github.com/labstack/echo/v4"
)

// IsAModel godoc
// @Summary Check if the given key is a RAS model
// @Description Check if the given key is a RAS model
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} bool
// @Router /isamodel [get]
func IsAModel(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		if filepath.Ext(definitionFile) != ".prj" {
			return c.JSON(http.StatusOK, false)
		}

		firstLine, err := tools.ReadFirstLine(*fs, definitionFile)
		if err != nil {
			return c.JSON(http.StatusOK, false)
		}
		if !strings.Contains(firstLine, "Proj Title=") {
			return c.JSON(http.StatusOK, false)
		}

		return c.JSON(http.StatusOK, true)
	}
}
