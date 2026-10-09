export type CityPreset = { id: string; name: string; latitude: number; longitude: number };

/** Manual location fallback when GPS is unavailable (coordinates are mandatory server-side). */
export const CITY_PRESETS: readonly CityPreset[] = [
  { id: 'paris', name: 'Paris', latitude: 48.86, longitude: 2.35 },
  { id: 'lyon', name: 'Lyon', latitude: 45.76, longitude: 4.84 },
  { id: 'marseille', name: 'Marseille', latitude: 43.3, longitude: 5.37 },
  { id: 'toulouse', name: 'Toulouse', latitude: 43.6, longitude: 1.44 },
  { id: 'nice', name: 'Nice', latitude: 43.7, longitude: 7.26 },
  { id: 'nantes', name: 'Nantes', latitude: 47.22, longitude: -1.55 },
  { id: 'bordeaux', name: 'Bordeaux', latitude: 44.84, longitude: -0.58 },
  { id: 'lille', name: 'Lille', latitude: 50.63, longitude: 3.06 },
  { id: 'strasbourg', name: 'Strasbourg', latitude: 48.57, longitude: 7.75 },
  { id: 'rennes', name: 'Rennes', latitude: 48.11, longitude: -1.68 },
  { id: 'bruxelles', name: 'Bruxelles', latitude: 50.85, longitude: 4.35 },
  { id: 'geneve', name: 'Genève', latitude: 46.2, longitude: 6.14 },
  { id: 'montreal', name: 'Montréal', latitude: 45.5, longitude: -73.57 },
];
