package handlers

import (
	"net/http"

	"github.com/USACE/filestore" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// ModelType godoc
// @Summary Extract the model type
// @Description Extract the model type given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {string} string "RAS"
// @Failure 500 {object} SimpleResponse
// @Router /modeltype [get]
func ModelType(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		if !isAModel(fs, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		return c.JSON(http.StatusOK, "RAS")
	}
}
