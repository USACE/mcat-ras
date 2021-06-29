-- MATERIALIZED VIEWS FOR MODELS

-- Project files
CREATE MATERIALIZED VIEW models.ras_project_metadata AS
SELECT
    models.model_inventory_id,
    c.collection_id AS collection,
    (models.model_metadata -> 'Metadata' -> 'ProjFileContents' ->> 'ProjTitle' ) AS title,
    (models.model_metadata -> 'Metadata' -> 'ProjFileContents' ->> 'Description') AS description,
    (models.model_metadata -> 'Metadata' -> 'ProjFileContents' ->> 'Units') AS units,
    (models.model_metadata -> 'Metadata' -> 'ProjFileContents' ->> 'CurrentPlan') AS current_plan,
    models.s3_key AS s3_key
FROM models.model AS models
LEFT JOIN inventory.collections AS c USING (collection_id)
WITH DATA;

-- Plan files 
CREATE MATERIALIZED VIEW models.ras_plan_metadata AS
with plan_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'Metadata' -> 'PlanFiles') as metadata
    FROM models.model
    WHERE (model_metadata -> 'Metadata' ->> 'PlanFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'PlanTitle') AS plan_title,
    (metadata ->> 'FileExt') AS file_ext,
    (metadata ->> 'ProgramVersion') AS version,
    (metadata ->> 'Description') AS description,
    (metadata ->> 'ShortIdentifier') AS short_id,
    (metadata ->> 'GeomFile') AS geom_file,
    (metadata ->> 'FlowFile') AS flow_file,
    (metadata ->> 'QuasiSteadyFile') AS quasi_steady_file,
    (metadata ->> 'UnsteadyFile') AS unsteady_file,
    (metadata ->> 'FlowRegime') AS flow_regime,
    (metadata ->> 'Path') AS s3_key
FROM plan_files
WITH DATA;
 
-- Flow files 
CREATE MATERIALIZED VIEW models.ras_flow_metadata AS
with flow_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'Metadata' -> 'FlowFiles') as metadata
    FROM models.model
    WHERE (model_metadata -> 'Metadata' ->> 'FlowFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'FlowTitle') AS flow_title,
    (metadata ->> 'FileExt') AS file_ext,
    (metadata ->> 'ProgramVersion') AS version,
    (metadata ->> 'NProfiles') AS num_profiles,
    (metadata ->> 'ProfileNames') AS profile_names,
    (metadata ->> 'Path') AS s3_key
FROM flow_files
WITH DATA;
 
-- Geometry files 
CREATE MATERIALIZED VIEW models.ras_geometry_metadata AS
with geom_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'Metadata' -> 'GeomFiles') as metadata
    FROM models.model
    WHERE (model_metadata -> 'Metadata' ->> 'GeomFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'Geom Title') AS geom_title,
    (metadata ->> 'File Extension') AS file_ext,
    (metadata ->> 'Program Version') AS version,
    (metadata ->> 'Description') AS description,
    json_array_length(CASE WHEN (metadata -> 'Hydraulic Structures')::text = 'null' THEN '[]'::json ELSE (metadata -> 'Hydraulic Structures') END) as num_reaches,
    (metadata ->> 'Path') AS s3_key
FROM geom_files
WITH DATA;
 
 
 -- Rivers
CREATE MATERIALIZED VIEW models.ras_rivers_metadata AS
with geom_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'Metadata' -> 'GeomFiles') as metadata
    FROM models.model
    WHERE (model_metadata -> 'Metadata' ->> 'GeomFiles') IS NOT NULL
),
hydraulic_structures as (
    SELECT
        model_inventory_id,
        json_array_elements(metadata -> 'Hydraulic Structures') as metadata
    FROM geom_files
    WHERE  (metadata ->> 'Hydraulic Structures') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'River Name') AS river_name,
    (metadata ->> 'Reach Name') AS reach_name,
    (metadata ->> 'Num CrossSections') AS num_xs,
    (metadata -> 'Culvert Data' ->> 'Num Culverts') AS num_culverts,
    (metadata-> 'Bridge Data' ->> 'Num Bridges') AS num_bridges,
    (metadata -> 'Inline Weir Data' ->> 'Num Inline Weirs') AS num_weirs
FROM hydraulic_structures
WITH DATA;

-- Convex Hull
CREATE MATERIALIZED VIEW models.ras_convexhull AS
SELECT 
    ras.model_inventory_id,
    ST_ConvexHull(ST_Union(ST_Force2D(xs.geom))) AS GEOM
FROM 
    models.ras_xs AS xs
INNER JOIN models.ras_rivers AS rivers USING (river_id)
INNER JOIN models.ras_geometry_files AS geom_files USING (geometry_file_id)
INNER JOIN models.model AS ras USING (model_inventory_id)
GROUP BY ras.model_inventory_id
WITH DATA;