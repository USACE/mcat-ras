package pgdb

import (
	"fmt"
	"os"
)

var (
	getCollectionIDSQL string = fmt.Sprintf(`
	SELECT collection_id 
	FROM inventory.collections 
	WHERE 's3://%s/'`, os.Getenv("S3_BUCKET")) + ` || $1 LIKE s3_prefix || '%';`

	getModelIDSQL string = `
		SELECT model_inventory_id 
		FROM models.model
		WHERE s3_key = $1;
		`

	getRiverIDSQL string = `
		SELECT river_id 
		FROM models.model
		WHERE geometry_file_id = $1 AND
		river_name = $2 AND
		reach_name = $3;
		`

	upsertModelSQL string = `
		INSERT INTO models.model (
			collection_id, 
			name,
			type, 
			s3_key, 
			model_metadata, 
			etl_metadata
			) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (s3_key)
		DO UPDATE SET 
			collection_id = $1, 
			name = $2, 
			type = $3, 
			s3_key = $4, 
			model_metadata = $5, 
			etl_metadata = $6
		RETURNING model_inventory_id;
	`

	upsertRiversSQL string = `
		INSERT INTO models.ras_rivers (
			geometry_file_id, 
			river_name, 
			reach_name, 
			geom
			) 
		VALUES ($1, $2, $3, ST_GeomFromWKB($4, 4326))
		ON CONFLICT (
			geometry_file_id,
			river_name,
			reach_name
		)
		DO UPDATE SET 
			geometry_file_id = $1, 
			river_name = $2, 
			reach_name = $3, 
			geom = ST_GeomFromWKB($4, 4326)
		RETURNING river_id;
	`

	upsertXSSQL string = `
		INSERT INTO models.ras_xs (
			river_id, 
			xs_station, 
			cut_line_profile_match, 
			geom
			) 
		VALUES ($1, $2, $3, ST_GeomFromWKB($4, 4326))
		ON CONFLICT (
			river_id,
			xs_station
		)
		DO UPDATE SET 
			river_id = $1, 
			xs_station = $2, 
			cut_line_profile_match = $3, 
			geom = ST_GeomFromWKB($4, 4326)
		RETURNING xs_id;
	`

	upsertBanksSQL string = `
		INSERT INTO models.ras_banks (
			xs_id, 
			bank_station, 
			geom) 
		VALUES ($1, $2, ST_GeomFromWKB($3, 4326))
		ON CONFLICT (
			xs_id,
			bank_station
		)
		DO UPDATE SET 
			xs_id = $1, 
			bank_station = $2, 
			geom = ST_GeomFromWKB($3, 4326);
	`

	upsertStorageAreasSQL string = `
		INSERT INTO models.ras_storage_areas (
			geometry_file_id, 
			storage_area_name, 
			geom
			) 
			VALUES ($1, $2, ST_GeomFromWKB($3, 4326))
		ON CONFLICT (storage_area_id)
		DO UPDATE SET 
			geometry_file_id = $1, 
			storage_area_name = $2, 
			geom = ST_GeomFromWKB($3, 4326);
	`

	upsertGeometrySQL string = `
		INSERT INTO models.ras_geometry_files (
			model_inventory_id, 
			geometry_file_path, 
			geometry_file_extension, 
			geometry_title, 
			geometry_program_version, 
			geometry_description
			) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (geometry_file_path)
		DO UPDATE SET 
			model_inventory_id = $1, 
			geometry_file_path = $2, 
			geometry_file_extension = $3, 
			geometry_title = $4, 
			geometry_program_version = $5, 
			geometry_description = $6 
		RETURNING geometry_file_id;
	`
)

// VacuumQuery ...
var vacuumQuery []string = []string{"VACUUM ANALYZE models.ras;",
	"VACUUM ANALYZE models.ras_geometry_files;",
	"VACUUM ANALYZE models.ras_rivers;",
	"VACUUM ANALYZE models.ras_xs;",
	"VACUUM ANALYZE models.ras_banks;",
	"VACUUM ANALYZE models.ras_storage_areas;",
	"VACUUM ANALYZE models.ras_two_d_areas;",
	"VACUUM ANALYZE models.ras_hydraulic_structures;"}

// RefreshViewsQuery ...
var refreshViewsQuery []string = []string{"REFRESH MATERIALIZED VIEW models.ras_project_metadata;",
	"REFRESH MATERIALIZED VIEW models.ras_plan_metadata;",
	"REFRESH MATERIALIZED VIEW models.ras_flow_metadata;",
	"REFRESH MATERIALIZED VIEW models.ras_geometry_metadata;",
	"REFRESH MATERIALIZED VIEW models.ras_rivers_metadata;",
	"REFRESH MATERIALIZED VIEW models.ras_convexhull;"}
