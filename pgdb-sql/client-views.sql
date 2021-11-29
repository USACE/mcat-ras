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
			
	 FROM ras_project_metadata t
	 JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
	 JOIN inventory.collections i ON i.collection_id = t.collection 
	) squery;

-- DROP VIEW models.ras_plan_files;
CREATE OR REPLACE VIEW models.ras_plan_files AS 

SELECT  squery.col_1 AS "1. Plan Title",
		squery.col_2 AS "2. Plan File",
 		squery.col_3 AS "3. Simulation Files (geometry | flow)",
 		squery.col_4 AS "4. Description", 
 		squery.col_5 AS "5. Flow Regime",
 		squery.col_6 AS "6. RAS Version",
 		squery.s3_key AS s3_key
FROM 
	(SELECT
		t.plan_title AS col_1,
		t.file_ext AS col_2,
		
	    CASE
           WHEN t.flow_file IS NOT NULL THEN  rgm.geom_title ||' | ' || rfm.flow_title
           WHEN t.unsteady_file IS NOT NULL THEN  rgm.geom_title  ||' | ' || rfm.flow_title
           WHEN t.quasi_steady_file IS NOT NULL THEN rgm.geom_title  ||' | ' || rfm.flow_title
	     END col_3,
	     
		t.description AS col_4, 
		t.flow_regime AS col_5,
		t."version" AS col_6, 
		r.s3_key AS s3_key
			
	 FROM ras_plan_metadata t
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
	 		squery.col_2 AS "2. Simulation File",
	 		squery.col_3 AS "3. Type", 
	 		squery.col_4 AS "4. Profiles",
	 		squery.col_5 AS "5. Profile Names",
	 		squery.col_6 AS "6. RAS Version",
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
				
		 FROM ras_flow_metadata t
		 JOIN models.model r ON r.model_inventory_id = t.model_inventory_id
		) squery;

-- TODO: SPlit this into 2 queries: for files and river-reaches
-- DROP VIEW models.ras_geometry_file_view;
CREATE VIEW models.ras_geometry_file_view AS 
SELECT geometry.geom_title AS "1. Geometry Title",
	geometry.file_ext AS "2. Simulation File",
    geometry.river_name || ' - ' || geometry.reach_name AS "3. River - Reach",
    geometry.geom_description AS "4. Description",
    geometry.n_cross_sections AS "5. Cross Sections",
    geometry.n_culverts AS "6. Culverts",
    geometry.num_bridges  AS "7. Bridges",
    geometry.num_inline_wiers  AS "8. Inline Wiers",
    geometry.geometry_model_version AS "9. Version", 
    plan.s3_key AS "s3_key"
    
   FROM ( SELECT c.title,
            m.name AS model_name,
            m.model_inventory_id,
            m.s3_key AS "s3_key"
           FROM models.model m
             JOIN inventory.collections c USING (collection_id)
          WHERE m.type = 'RAS'::text AND (m.model_metadata ->> 'PlanFiles'::text) <> 'null'::text 
          AND (m.model_metadata ->> 'ProjFileContents'::text) <> 'null'::text) plan
          
          
     JOIN ( WITH query_1 AS (
                 SELECT m.model_inventory_id,
                    m.model_metadata ->> 'GeomFiles'::text AS geom_files
                   FROM models.model m
                  ORDER BY m.model_inventory_id
                ), query_2 AS (
                 SELECT query_1.model_inventory_id,
                    json_array_elements(query_1.geom_files::json) ->> 'Program Version'::text AS geometry_model_version,
                    json_array_elements(query_1.geom_files::json) ->> 'Geom Title'::text AS geom_title,
                    json_array_elements(query_1.geom_files::json) ->> 'File Extension'::text AS file_ext,
                    json_array_elements(query_1.geom_files::json) ->> 'Description'::text AS geom_description,
                    json_array_elements(query_1.geom_files::json) ->> 'Hydraulic Structures'::text AS structs
                   FROM query_1
                  WHERE query_1.geom_files IS NOT NULL
                  ORDER BY query_1.model_inventory_id
                )
                
         SELECT query_2.model_inventory_id,
            query_2.geometry_model_version,
            query_2.geom_title,
            query_2.file_ext,
            query_2.geom_description,
            (json_array_elements(query_2.structs::json) -> 'Inline Weir Data'::text) ->> 'Num Inline Weirs'::text AS num_inline_wiers,
            (json_array_elements(query_2.structs::json) -> 'Culvert Data'::text) ->> 'Num Culverts'::text AS n_culverts,
            (json_array_elements(query_2.structs::json) -> 'Bridge Data'::text) ->> 'Num Bridges'::text AS num_bridges,
            json_array_elements(query_2.structs::json) ->> 'Num CrossSections'::text AS n_cross_sections,
            json_array_elements(query_2.structs::json) ->> 'Reach Name'::text AS reach_name,
            json_array_elements(query_2.structs::json) ->> 'River Name'::text AS river_name
           FROM query_2
          WHERE query_2.structs IS NOT NULL
          ORDER BY query_2.geom_title) geometry USING (model_inventory_id)

  ORDER BY "1. Geometry Title";