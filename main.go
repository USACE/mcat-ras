package main

// @title RAS MCAT API
// @version 1.0
// @description API for the RAS MCAT
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email slawler@dewberry.com
// @host localhost:5600

import (
	"github.com/USACE/mcat-ras/config"
	_ "github.com/USACE/mcat-ras/docs"
	"github.com/USACE/mcat-ras/handlers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {

	// Connect to backend services
	appConfig := config.Init()

	// Instantiate echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// HealthCheck
	e.GET("/ping", handlers.Ping(appConfig.FileStore))

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// ras endpoints
	e.GET("/index", handlers.Index(appConfig.FileStore))
	e.GET("/isamodel", handlers.IsAModel(appConfig.FileStore))
	e.GET("/isgeospatial", handlers.IsGeospatial(appConfig.FileStore))
	e.GET("/modeltype", handlers.ModelType(appConfig.FileStore))
	e.GET("/modelversion", handlers.ModelVersion(appConfig.FileStore))
	e.GET("/geospatialdata", handlers.GeospatialData(appConfig.FileStore, appConfig.DestinationCRS))

	e.Logger.Fatal(e.Start(appConfig.Address()))
}
