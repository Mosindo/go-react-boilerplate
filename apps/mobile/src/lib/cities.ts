// Fallback when the device location is unavailable or refused. Coordinates are
// city centres, so choosing one reveals nothing more precise than the city itself.
export type City = { name: string; latitude: number; longitude: number };

export const CITIES: readonly City[] = [
  { name: "Paris", latitude: 48.8566, longitude: 2.3522 },
  { name: "Marseille", latitude: 43.2965, longitude: 5.3698 },
  { name: "Lyon", latitude: 45.764, longitude: 4.8357 },
  { name: "Toulouse", latitude: 43.6047, longitude: 1.4442 },
  { name: "Nice", latitude: 43.7102, longitude: 7.262 },
  { name: "Nantes", latitude: 47.2184, longitude: -1.5536 },
  { name: "Montpellier", latitude: 43.6108, longitude: 3.8767 },
  { name: "Strasbourg", latitude: 48.5734, longitude: 7.7521 },
  { name: "Bordeaux", latitude: 44.8378, longitude: -0.5792 },
  { name: "Lille", latitude: 50.6292, longitude: 3.0573 },
  { name: "Rennes", latitude: 48.1173, longitude: -1.6778 },
  { name: "Grenoble", latitude: 45.1885, longitude: 5.7245 },
  { name: "Bruxelles", latitude: 50.8503, longitude: 4.3517 },
  { name: "Genève", latitude: 46.2044, longitude: 6.1432 },
  { name: "Montréal", latitude: 45.5017, longitude: -73.5673 },
  { name: "Londres", latitude: 51.5072, longitude: -0.1276 },
  { name: "Madrid", latitude: 40.4168, longitude: -3.7038 },
  { name: "Berlin", latitude: 52.52, longitude: 13.405 },
  { name: "Casablanca", latitude: 33.5731, longitude: -7.5898 },
  { name: "Dakar", latitude: 14.7167, longitude: -17.4677 }
];

export function searchCities(query: string): readonly City[] {
  const normalize = (v: string) => v.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase().trim();
  const q = normalize(query);
  if (!q) {
    return CITIES;
  }
  return CITIES.filter((city) => normalize(city.name).includes(q));
}
