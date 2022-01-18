package pgdb

import (
	"app/config"
	ras "app/tools"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/jmoiron/sqlx"
)

type ETLMetaData struct {
	ModelName            string `json:"model_name"`
	SourcePath           string `json:"source_path"`
	DestinationPath      string `json:"destination_path"`
	ProjectionSourcePath string `json:"projection_source_path"`
}

func getCollectionID(tx *sqlx.Tx, definitionFile string) (collectionID int, err error) {

	if err := tx.Get(&collectionID, getCollectionIDSQL, definitionFile); err != nil {
		return 0, errors.Wrap(err, 0)
	}
	return collectionID, nil
}

func getModelID(tx *sqlx.Tx, definitionFile string) (modelID int, err error) {
	if err := tx.Get(&modelID, getModelIDSQL, definitionFile); err != nil {
		return 0, errors.Wrap(err, 0)
	}
	return modelID, nil
}

func upsertModel(tx *sqlx.Tx, rm *ras.RasModel, definitionFile string, collectionID int) (modelID int, err error) {
	projFileName := filepath.Base(definitionFile)
	modelName := strings.TrimSuffix(projFileName, filepath.Ext(projFileName))

	etlMetaRaw := ETLMetaData{ModelName: modelName, SourcePath: definitionFile}

	etlMeta, err := json.Marshal(etlMetaRaw)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	modelMeta, err := json.Marshal(rm.Metadata)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	if err := tx.Get(&modelID, upsertModelSQL, collectionID, modelName, rm.Type, definitionFile, modelMeta, etlMeta); err != nil {
		return 0, errors.Wrap(err, 0)
	}

	return modelID, nil
}

func upsertRiver(tx *sqlx.Tx, river ras.VectorLayer, geometryFileID int) (riverID int, err error) {
	riverReachName := river.FeatureName
	riverReach := strings.Split(riverReachName, ",")
	riverName := strings.TrimSpace(riverReach[0])
	reachName := strings.TrimSpace(riverReach[1])

	if err := tx.Get(&riverID, upsertRiversSQL, geometryFileID, riverName, reachName, river.Geometry); err != nil {
		return 0, errors.Wrap(err, 0)
	}
	return riverID, nil
}

func upsertModelInfo(definitionFile string, ac *config.APIConfig, db *sqlx.DB) error {
	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Println(err)
		return errors.Wrap(err, 0)
	}

	rm, err := ras.NewRasModel(definitionFile, *ac.FileStore)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	collectionID, err := getCollectionID(tx, definitionFile)
	if err != nil {
		log.Println("Collection ID:", collectionID, err)
		return errors.Wrap(err, 0)
	}

	modelID, err := upsertModel(tx, rm, definitionFile, collectionID)
	if err != nil {
		fmt.Println("Model ID:", modelID, "Name|", definitionFile)
		log.Println("Error: ", err, "Rolling back")
		tx.Rollback()
		return errors.Wrap(err, 0)
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("Model ID:", modelID, "Name|", definitionFile)
		log.Println("Transaction Commit Error|", err)
		return errors.Wrap(err, 0)
	}

	return nil
}

func upsertModelGeometry(definitionFile string, ac *config.APIConfig, db *sqlx.DB) error {
	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Println(err)
		return errors.Wrap(err, 0)
	}

	rm, err := ras.NewRasModel(definitionFile, *ac.FileStore)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	modelID, err := getModelID(tx, definitionFile)
	fmt.Println("Model ID:", modelID, "Name|", definitionFile)
	if err != nil {
		log.Println(err)
		return errors.Wrap(err, 0)
	}

	if rm.IsGeospatial() {

		geodata, err := rm.GeospatialData(ac.DestinationCRS)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// Iterate over geometry files
		for _, geometryFile := range rm.Metadata.GeomFiles {
			var geometryFileID int

			var version interface{} = geometryFile.ProgramVersion
			if geometryFile.ProgramVersion == "" {
				version = sql.NullFloat64{Float64: 0.0, Valid: false}
			} // doing this to prevent SQL error when inserting "" to a numeric field

			// Add Geometry file to database
			if err = tx.Get(&geometryFileID, upsertGeometrySQL,
				modelID,
				geometryFile.Path,
				geometryFile.FileExt,
				geometryFile.GeomTitle,
				version,
				geometryFile.Description); err != nil {
				log.Println("Geometry File", geometryFile.FileExt, "|", err)
				tx.Rollback()
				return errors.Wrap(err, 0)
			}

			// Iterate over features in geometry file and add to tables as needed
			geomFileName := filepath.Base(geometryFile.Path)
			features := geodata.Features[geomFileName]

			// Create Dynamic container to map rivers/reaches with xs/banks
			riverIDMap := make(map[string]int, len(features.Rivers))

			// Add all rivers
			for _, river := range features.Rivers {
				riverID, err := upsertRiver(tx, river, geometryFileID)
				if err != nil {
					log.Println(err)
					tx.Rollback()
					return errors.Wrap(err, 0)
				}
				riverIDMap[river.FeatureName] = riverID
			}

			// Add all XS
			xsIDMap := make(map[string]int, len(features.XS))
			for _, xs := range features.XS {
				var xsID int
				riverReach := xs.Fields["RiverReachName"]
				cutLineProfileMatch := xs.Fields["CutLineProfileMatch"]
				xsStation, err := strconv.ParseFloat(xs.FeatureName, 64)
				if err != nil {
					log.Println("XS", geometryFile.FileExt, "|", err)
					tx.Rollback()
					return errors.Wrap(err, 0)
				}

				riverID := riverIDMap[riverReach.(string)]
				if err = tx.Get(&xsID, upsertXSSQL, riverID, xsStation, cutLineProfileMatch, xs.Geometry); err != nil {
					log.Println(err)
					tx.Rollback()
					return errors.Wrap(err, 0)
				}
				riverReachXSName := fmt.Sprintf("%s-%s", riverReach, xs.FeatureName)
				xsIDMap[riverReachXSName] = xsID
			}

			// Add all Banks
			for _, banks := range features.Banks {
				riverReachXSName := fmt.Sprintf("%s-%s", banks.Fields["RiverReachName"], banks.Fields["xsName"].(string))
				xsID := xsIDMap[riverReachXSName]
				bankStation, err := strconv.ParseFloat(banks.FeatureName, 64)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				_, err = tx.Exec(upsertBanksSQL, xsID, bankStation, banks.Geometry)
				if err != nil {
					log.Println("Banks", geometryFile.FileExt, "|", err)
					tx.Rollback()
					return errors.Wrap(err, 0)
				}
			}

			// Add all Storage Areas
			for _, storageArea := range features.StorageAreas {
				_, err = tx.Exec(upsertStorageAreasSQL, geometryFileID, storageArea.FeatureName, storageArea.Geometry)
				if err != nil {
					log.Println("Storage Areas", geometryFile.FileExt, "|", err)
					tx.Rollback()
					return errors.Wrap(err, 0)
				}
			}
		}
		
		err = tx.Commit()
		if err != nil {
			log.Println("Transaction Commit Error|", err)
			return errors.Wrap(err, 0)
		}

	}
	return nil
}
