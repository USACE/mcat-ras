package handlers

import (
	ras "app/tools"
	"net/http"

	"github.com/USACE/filestore"
	"github.com/labstack/echo/v4"
)

// ModelType godoc
// @Summary Extract the model type
// @Description Extract the model type given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/pfra-models/mipmodels/MD/M000309/T1ChptnkR.prj"
// @Success 200 {string} string "RAS"
// @Failure 500 {object} SimpleResponse
// @Router /modeltype [get]
func ModelType(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		rm, err := ras.NewRasModel(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, err.Error()})
		}
		typ := rm.ModelType()

		return c.JSON(http.StatusOK, typ)
	}
}
