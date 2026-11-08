-- Great-circle distance in kilometres (haversine). Pure SQL, no PostGIS dependency.
CREATE OR REPLACE FUNCTION geo_distance_km(lat1 DOUBLE PRECISION, lng1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lng2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 2 * 6371 * asin(least(1, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lng2 - lng1) / 2), 2)
  )))
$$;
