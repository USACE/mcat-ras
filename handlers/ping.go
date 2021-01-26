package handlers

import (
	"fmt"
	"net/http"

	"github.com/USACE/filestore"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/labstack/echo/v4"
)

type SimpleResponse struct {
	Status  int
	Message string
}

// Ping godoc
// @Summary Status Check
// @Description Check which services are operational
// @Tags Health Check
// @Accept  json
// @Produce  json
// @Success 200 {object} SimpleResponse
// @Router /ping [get]
func Ping(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch (*fs).(type) {
		case *filestore.BlockFS:
			fmt.Println("File is local")
			return c.JSON(http.StatusOK, SimpleResponse{http.StatusOK, "🏓 Pong! 🏓"})

		case *filestore.S3FS:
			s3FS := (*fs).(*filestore.S3FS)
			err := s3FS.Ping()
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() != "AccessDenied" {
						msg := fmt.Sprintf("Unavailable: %s", awsErr.Code())
						return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, msg})
					}
				} else {
					return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, "Unavailable"})
				}
			}
			fmt.Println("File is on S3")
			return c.JSON(http.StatusOK, SimpleResponse{http.StatusOK, "🏓 Pong! 🏓"})
		}
		return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, "Unavailable"})
	}
}
