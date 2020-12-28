package handlers

import (
	"net/http"

	ras "github.com/USACE/mcat-ras/tools"

	"github.com/USACE/filestore"
	"github.com/labstack/echo/v4"
)

// IsGeospatial godoc
// @Summary Check if the RAS model has geospatial information
// @Description  Check if the RAS model has geospatial information
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/pfra-models/mipmodels/MD/M000309/T1ChptnkR.prj"
// @Success 200 {object} bool
// @Router /isgeospatial [get]
func IsGeospatial(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		rm, err := ras.NewRasModel(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusOK, false)
		}
		isIt := rm.IsGeospatial()

		return c.JSON(http.StatusOK, isIt)
	}
}
