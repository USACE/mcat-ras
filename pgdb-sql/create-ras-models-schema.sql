CREATE SCHEMA IF NOT EXISTS models;

/*---------------------------------------------------------------------------*/
-- Create models.column_reference table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.column_reference(
    model_type TEXT PRIMARY KEY,
    map JSON NOT NULL
);


/*---------------------------------------------------------------------------*/
-- Create models.model table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.model (
    model_inventory_id BIGINT PRIMARY KEY,
    collection_id BIGINT,
    name TEXT,
    type TEXT,
    s3_key TEXT UNIQUE NOT NULL,
    model_metadata JSON NOT NULL,
    etl_metadata JSON NOT NULL,
    CONSTRAINT model_collection_id_fk FOREIGN KEY (collection_id) REFERENCES inventory.collections (collection_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT model_type_check CHECK (
        type = 'RAS' OR
        type = 'HMS' OR
        type = 'OTHER')
);

-- Create indexes on foreign keys
CREATE INDEX IF NOT EXISTS collection_id_idx ON models.model (collection_id);
CREATE INDEX IF NOT EXISTS model_type_idx ON models.model (type);


/*---------------------------------------------------------------------------*/
-- Create models.ras_geometry_files table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_geometry_files(
       geometry_file_id SERIAL PRIMARY KEY,
       model_inventory_id INTEGER REFERENCES models.model ON UPDATE CASCADE ON DELETE CASCADE,
       geometry_file_path TEXT NOT NULL UNIQUE,
       geometry_file_extension TEXT NOT NULL,
       geometry_title TEXT NOT NULL,
       geometry_program_version DECIMAL,
       geometry_description TEXT
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_geometry_files_ras_fk_idx ON models.ras_geometry_files (model_inventory_id);


/*---------------------------------------------------------------------------*/
-- Create models.ras_rivers table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_rivers(
       river_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       river_name TEXT NOT NULL,
       reach_name TEXT NOT NULL,
       geom GEOMETRY(MultiLineString, 4326),
       CONSTRAINT ras_rivers_geomfile_river_reach UNIQUE (geometry_file_id, river_name, reach_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_rivers_geometry_file_id_idx ON models.ras_rivers (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_rivers_geom_idx ON models.ras_rivers USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_xs table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_xs(
       xs_id SERIAL PRIMARY KEY,
       river_id INTEGER REFERENCES models.ras_rivers ON UPDATE CASCADE ON DELETE CASCADE,
       xs_station DECIMAL NOT NULL,
       geom GEOMETRY(MultiLineStringZ, 4326),
       cut_line_profile_match BOOLEAN NOT NULL,
       CONSTRAINT ras_xs_river_fk_xs_station UNIQUE (river_id, xs_station)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_xs_river_id_idx ON models.ras_xs (river_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_xs_geom_idx ON models.ras_xs USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_banks table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_banks(
      bank_id SERIAL PRIMARY KEY,
      xs_id INTEGER REFERENCES models.ras_xs ON UPDATE CASCADE ON DELETE CASCADE,
      bank_station DECIMAL NOT NULL,
      geom GEOMETRY(MultiPoint, 4326),
      CONSTRAINT ras_banks_xs_id_bank_station UNIQUE (xs_id, bank_station)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_banks_xs_id_idx ON models.ras_banks (xs_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_banks_geom_idx ON models.ras_banks USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_storage_areas table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_storage_areas(
       storage_area_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       storage_area_name TEXT NOT NULL,
       geom GEOMETRY(MultiPolygon, 4326),
       CONSTRAINT ras_storage_areas_geometry_file_id_name_uniq UNIQUE (geometry_file_id, storage_area_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_storage_areas_geometry_file_id_idx ON models.ras_storage_areas (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_storage_areas_geom_idx ON models.ras_storage_areas USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_two_d_areas table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_two_d_areas(
       two_d_area_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       two_d_area_name TEXT NOT NULL,
       geom GEOMETRY(MultiPolygon, 4326),
       CONSTRAINT ras_two_d_areas_geometry_file_id_name_uniq UNIQUE (geometry_file_id, two_d_area_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_two_d_areas_geometry_file_id_idx ON models.ras_two_d_areas (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_two_d_areas_geom_idx ON models.ras_two_d_areas USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_hydraulic_structures table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_hydraulic_structures(
       hydraulic_structure_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       hydraulic_structure_name TEXT NOT NULL,
       hydraulic_structure_type TEXT NOT NULL,
       geom GEOMETRY(MultiLineString, 4326),
       CONSTRAINT geometry_file_id_hydraulic_structure_name_hydraulic_structure_type_uniq UNIQUE (geometry_file_id, hydraulic_structure_name, hydraulic_structure_type)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_hydraulic_structures_geometry_file_id_idx ON models.ras_hydraulic_structures (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_hydraulic_structure_geom_idx ON models.ras_hydraulic_structures USING GIST (geom);