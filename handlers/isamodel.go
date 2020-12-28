package handlers

import (
	ras "app/tools"
	"net/http"

	"github.com/USACE/filestore"
	"github.com/labstack/echo/v4"
)

// IsAModel godoc
// @Summary Check if the given key is a RAS model
// @Description Check if the given key is a RAS model
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/pfra-models/mipmodels/MD/M000309/T1ChptnkR.prj"
// @Success 200 {object} bool
// @Router /isamodel [get]
func IsAModel(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		rm, err := ras.NewRasModel(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusOK, false)
		}
		isIt := rm.IsAModel()

		return c.JSON(http.StatusOK, isIt)
	}
}
