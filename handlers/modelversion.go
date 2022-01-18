package handlers

import (
	"fmt"
	"net/http"

	ras "app/tools"

	"github.com/USACE/filestore"
	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// ModelVersion godoc
// @Summary Extract the RAS model version
// @Description Extract the RAS model version given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {string} string "4.0"
// @Failure 500 {object} SimpleResponse
// @Router /modelversion [get]
func ModelVersion(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		rm, err := ras.NewRasModel(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, fmt.Sprintf("Go error encountered: %v", err.Error()), err.(*errors.Error).ErrorStack()})
		}
		vers := rm.ModelVersion()

		return c.JSON(http.StatusOK, vers)
	}
}
