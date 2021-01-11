package handlers

import (
	"net/http"

	"github.com/USACE/mcat-ras/config"
	ras "github.com/USACE/mcat-ras/tools"

	"github.com/labstack/echo/v4"
)

// GeospatialData godoc
// @Summary Extract geospatial data
// @Description Extract geospatial data from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/pfra-models/mipmodels/MD/M000309/T1ChptnkR.prj"
// @Success 200 {object} interface{}
// @Failure 500 {object} SimpleResponse
// @Router /geospatialdata [get]
func GeospatialData(ac *config.APIConfig) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		rm, err := ras.NewRasModel(definitionFile, *ac.FileStore)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, err.Error()})
		}

		data, err := rm.GeospatialData(ac.DestinationCRS)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, err.Error()})
		}

		return c.JSON(http.StatusOK, data)
	}
}
