package pgdb

import (
	"app/config"
	"app/handlers"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// UpsertRasModel ...
func UpsertRasModel(ac *config.APIConfig, db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest,
				handlers.SimpleResponse{Status: http.StatusBadRequest,
					Message: "Missing query parameter: `definition_file`"})
		}

		err := upsertModelInfo(definitionFile, ac, db)
		if err != nil {
			return c.JSON(http.StatusNotAcceptable, err)
		}

		return c.JSON(http.StatusOK, "Successfully uploaded model information for "+definitionFile)
	}
}

// UpsertRasGeometry ...
func UpsertRasGeometry(ac *config.APIConfig, db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")

		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest,
				handlers.SimpleResponse{Status: http.StatusBadRequest,
					Message: "Missing query parameter: `definition_file`"})
		}

		err := upsertModelGeometry(definitionFile, ac, db)
		if err != nil {
			return c.JSON(http.StatusNotAcceptable, err)
		}

		return c.JSON(http.StatusOK, "Successfully uploaded model geometry for "+definitionFile)
	}
}

// VacuumRasViews ...
func VacuumRasViews(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		for _, query := range vacuumQuery {
			_, err := db.Exec(query)
			if err != nil {
				return c.JSON(http.StatusNotAcceptable, err)
			}
		}

		return c.JSON(http.StatusOK, "Ras tables vacuumed successfully.")
	}
}

// RefreshRasViews ...
func RefreshRasViews(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		for _, query := range refreshViewsQuery {
			_, err := db.Exec(query)
			if err != nil {
				return c.JSON(http.StatusNotAcceptable, err)
			}
		}

		return c.JSON(http.StatusOK, "Ras materialized views refreshed successfully.")
	}
}
