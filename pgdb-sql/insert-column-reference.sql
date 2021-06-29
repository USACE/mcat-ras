INSERT INTO models.column_reference (model_type, map)
VALUES (:col_ref_type, :col_ref_json)
ON CONFLICT (model_type) DO UPDATE SET map = EXCLUDED.map;