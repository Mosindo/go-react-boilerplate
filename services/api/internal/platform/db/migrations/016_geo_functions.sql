-- Great-circle distance in kilometres (haversine). Used by discovery filters.
CREATE OR REPLACE FUNCTION geo_distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT 6371.0 * 2.0 * asin(LEAST(1.0, sqrt(
    power(sin(radians(lat2 - lat1) / 2.0), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2.0), 2)
  )))
$$;
