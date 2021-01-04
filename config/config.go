package config

import (
	"fmt"
	"os"

	"github.com/USACE/filestore"
)

type APIConfig struct {
	Host           string
	Port           int
	FileStore      *filestore.FileStore
	DestinationCRS int
}

// Address tells the application where to run the api out of
func (app *APIConfig) Address() string {
	return fmt.Sprintf("%s:%d", app.Host, app.Port)
}

// Init initializes the API's configuration
func Init() *APIConfig {
	config := new(APIConfig)
	config.Host = "" // 0.0.0.0
	config.Port = 5600
	config.FileStore = FileStoreInit(false)
	config.DestinationCRS = 4326
	return config
}

// FileStoreInit initializes the filestore object
func FileStoreInit(local bool) *filestore.FileStore {

	var fs filestore.FileStore
	var err error
	switch local {
	case true:
		fs, err = filestore.NewFileStore(filestore.BlockFSConfig{})
		if err != nil {
			panic(err)
		}
	case false:
		config := filestore.S3FSConfig{
			S3Id:     os.Getenv("AWS_ACCESS_KEY_ID"),
			S3Key:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
			S3Region: os.Getenv("AWS_DEFAULT_REGION"),
			S3Bucket: os.Getenv("S3_BUCKET"),
		}

		fs, err = filestore.NewFileStore(config)
		if err != nil {
			panic(err)
		}
	}
	return &fs
}
