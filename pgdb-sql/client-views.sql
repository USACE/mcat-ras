-- Check views for upstream tables/materialized views before changing/using!
--DROP VIEW models.ras_project_summary;
CREATE OR REPLACE VIEW models.ras_project_summary AS 

SELECT  squery.col_1 AS "1. Project Title",
 		squery.col_2 AS "2. Description", 
 		squery.col_3 AS "3. Units",
 		squery.col_4 AS "4. Data Collection",
 		squery.col_5 AS "5. Source",
 		squery.s3_key AS s3_key
FROM 
	(SELECT
		t.title AS col_1,
		t.description AS col_2,
		t.units AS col_3, 
		i.title AS col_4,
		i."source" AS col_5,
		r.s3_key AS s3_key
			
	 FROM models.ras_project_metadata t
	 JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
	 JOIN inventory.collections i ON i.collection_id = t.collection 
	) squery;

-- DROP VIEW models.ras_plan_files;
CREATE OR REPLACE VIEW models.ras_plan_files AS 

SELECT  squery.col_1 AS "1. Plan Title",
		squery.col_2 AS "2. File Ext",
 		squery.col_3 AS "3. Simulation Files (geometry | flow)",
 		squery.col_4 AS "4. Description", 
 		squery.col_5 AS "5. Flow Regime",
 		squery.col_6 AS "6. Version",
 		squery.s3_key AS s3_key
FROM 
	(SELECT
		t.plan_title AS col_1,
		t.file_ext AS col_2,
		
	    CASE
           WHEN t.flow_file IS NOT NULL THEN  rgm.file_ext || ' | ' || rfm.file_ext
           WHEN t.unsteady_file IS NOT NULL THEN  rgm.file_ext  ||' | ' || rfm.file_ext
           WHEN t.quasi_steady_file IS NOT NULL THEN rgm.file_ext  ||' | ' || rfm.file_ext
	     END col_3,
	     
		t.description AS col_4, 
		t.flow_regime AS col_5,
		t."version" AS col_6, 
		r.s3_key AS s3_key
			
	 FROM models.ras_plan_metadata t
	 JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
	 LEFT JOIN models.ras_geometry_metadata rgm 
	 	ON rgm.model_inventory_id = t.model_inventory_id
	 	AND t.geom_file = LTRIM(rgm.file_ext,'.')
	 LEFT JOIN models.ras_flow_metadata rfm 
	 	ON rfm.model_inventory_id = t.model_inventory_id
	 	AND t.flow_file = LTRIM(rfm.file_ext,'.')
	) squery;

-- DROP VIEW models.ras_flow_files;
CREATE OR REPLACE VIEW models.ras_flow_files AS 

	SELECT  squery.col_1 AS "1. Flow Title",
	 		squery.col_2 AS "2. File Ext",
	 		squery.col_3 AS "3. Type", 
	 		squery.col_4 AS "4. Num Profiles",
	 		squery.col_5 AS "5. Profile Names",
	 		squery.col_6 AS "6. Version",
	 		squery.s3_key AS s3_key
	FROM 
		(SELECT
			t.flow_title AS col_1, 
			t.file_ext AS col_2,
		    CASE

            -- update tjios!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	           WHEN t.file_ext IS NOT NULL THEN 'Steady'
	           WHEN t.file_ext IS NOT NULL THEN 'Unsteady'
	           WHEN t.file_ext IS NOT NULL THEN 'Quasi-Steady'
		     END col_3,
		     
			t.num_profiles AS col_4, 
			t.profile_names AS col_5,
			t."version" AS col_6, 
			r.s3_key AS s3_key
				
		 FROM models.ras_flow_metadata t
		 JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
		) squery;

-- DROP VIEW models.ras_geometry_files_view;
CREATE OR REPLACE VIEW models.ras_geometry_files_view AS 
	SELECT 	squery.geom_title AS "1. Geometry Title",
			squery.file_ext AS "2. File Ext",
			squery.description AS "3. Description",
			squery.version AS "4. Version",
			squery.num_reaches AS "5. Num Reaches", 
			squery.num_storage_areas AS "6. Num Storage Areas", 
			squery.num_two_d_areas AS "7. Num 2D Areas",
			squery.num_connections AS "8. Num Connections", 
			squery.s3_key AS "s3_key"
	FROM 
		(SELECT
			t.geom_title,
			t.file_ext,
			t.description,
			t.version,
			t.num_reaches,
			t.num_storage_areas,
			t.num_two_d_areas,
			t.num_connections,
			r.s3_key

			FROM models.ras_geometry_metadata t
			JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
		) squery
	ORDER BY "2. File Ext";

-- Ras Rivers View
CREATE OR REPLACE VIEW models.ras_rivers_view AS
	SELECT 	t.river_name AS "1. River",
			t.reach_name AS "2. Reach",
			t.num_xs AS "3. Num Cross Sections",
			t.num_culverts AS "4. Num Culverts",
			t.num_bridges  AS "5. Num Bridges",
			t.num_weirs  AS "6. Num Inline Weirs",
			t.file_ext,
			t.s3_key
	FROM models.ras_rivers_metadata t
	ORDER BY "1. River";