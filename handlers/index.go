package handlers

import (
	"net/http"

	ras "github.com/USACE/mcat-ras/tools"

	"github.com/USACE/filestore"
	"github.com/labstack/echo/v4"
)

// Index godoc
// @Summary Index a RAS model
// @Description Extract metadata from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} ras.Model
// @Failure 500 {object} SimpleResponse
// @Router /index [get]
func Index(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		rm, err := ras.NewRasModel(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, err.Error()})
		}
		mod := rm.Index()

		return c.JSON(http.StatusOK, mod)
	}
}
