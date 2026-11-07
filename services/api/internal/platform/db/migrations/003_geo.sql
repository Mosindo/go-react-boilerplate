-- Great-circle distance in km (haversine). Used for discovery filtering and distance display.
CREATE OR REPLACE FUNCTION geo_distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION AS $$
  SELECT 12742 * asin(LEAST(1, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$ LANGUAGE SQL IMMUTABLE STRICT;
