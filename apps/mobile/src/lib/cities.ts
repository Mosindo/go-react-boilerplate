export type City = { name: string; latitude: number; longitude: number };

/** Fallback when the device position is unavailable or refused. Coordinates are city centres. */
export const CITIES: City[] = [
  { name: "Paris", latitude: 48.86, longitude: 2.35 },
  { name: "Lyon", latitude: 45.76, longitude: 4.84 },
  { name: "Marseille", latitude: 43.3, longitude: 5.37 },
  { name: "Toulouse", latitude: 43.6, longitude: 1.44 },
  { name: "Nice", latitude: 43.7, longitude: 7.26 },
  { name: "Nantes", latitude: 47.22, longitude: -1.55 },
  { name: "Bordeaux", latitude: 44.84, longitude: -0.58 },
  { name: "Lille", latitude: 50.63, longitude: 3.06 },
  { name: "Strasbourg", latitude: 48.57, longitude: 7.75 },
  { name: "Rennes", latitude: 48.11, longitude: -1.68 },
  { name: "Bruxelles", latitude: 50.85, longitude: 4.35 },
  { name: "Genève", latitude: 46.2, longitude: 6.14 },
  { name: "Montréal", latitude: 45.5, longitude: -73.57 }
];
