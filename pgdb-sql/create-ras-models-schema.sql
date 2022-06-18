CREATE SCHEMA IF NOT EXISTS models;

/*---------------------------------------------------------------------------*/
-- Create models.model table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.model (
    model_inventory_id BIGSERIAL PRIMARY KEY,
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
-- Create models.ras_areas table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_areas(
       area_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       area_name TEXT NOT NULL,
       is2d BOOLEAN NOT NULL,
       geom GEOMETRY(MultiPolygon, 4326),
       CONSTRAINT ras_areas_geometry_file_id_name_uniq UNIQUE (geometry_file_id, area_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_areas_geometry_file_id_idx ON models.ras_areas (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_areas_geom_idx ON models.ras_areas USING GIST (geom);


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

/*---------------------------------------------------------------------------*/
-- Create models.ras_connections table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_connections(
       connection_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       connection_name TEXT NOT NULL,
       up_area TEXT NOT NULL,
       dn_area TEXT NOT NULL,
       geom GEOMETRY(MultiLineString, 4326),
       CONSTRAINT ras_connections_geometry_file_id_name_uniq UNIQUE (geometry_file_id, connection_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_connections_geometry_file_id_idx ON models.ras_connections (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_connections_geom_idx ON models.ras_connections USING GIST (geom);


/*---------------------------------------------------------------------------*/
-- Create models.ras_bclines table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_bclines(
       bcline_id SERIAL PRIMARY KEY,
       area_id INTEGER REFERENCES models.ras_areas ON UPDATE CASCADE ON DELETE CASCADE,
       bcline_name TEXT NOT NULL,
       geom GEOMETRY(MultiLineString, 4326),
       CONSTRAINT ras_bclines_file_id_name_uniq UNIQUE (area_id, bcline_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_bclines_areas_id_idx ON models.ras_bclines (area_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_bclines_geom_idx ON models.ras_bclines USING GIST (geom);



/*---------------------------------------------------------------------------*/
-- Create models.ras_breaklines table
/*---------------------------------------------------------------------------*/
CREATE TABLE IF NOT EXISTS models.ras_breaklines(
       breakline_id SERIAL PRIMARY KEY,
       geometry_file_id INTEGER REFERENCES models.ras_geometry_files ON UPDATE CASCADE ON DELETE CASCADE,
       breakline_name TEXT NOT NULL,
       geom GEOMETRY(MultiLineString, 4326),
       CONSTRAINT ras_breaklines_geomfile_file_id_name_uniq UNIQUE (geometry_file_id, breakline_name)
);

-- Create index on foreign key
CREATE INDEX IF NOT EXISTS ras_rivers_geometry_file_id_idx ON models.ras_breaklines (geometry_file_id);

-- Create index on geometry
CREATE INDEX IF NOT EXISTS ras_breaklines_geom_idx ON models.ras_breaklines USING GIST (geom);