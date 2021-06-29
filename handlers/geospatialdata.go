package handlers

import (
	"net/http"

	"app/config"

	ras "app/tools"

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
