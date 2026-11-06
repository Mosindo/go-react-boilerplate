// Fallback for people who prefer not to share GPS: a city centre is enough for
// matching and is naturally approximate.
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
  { name: "Montréal", latitude: 45.5019, longitude: -73.5674 }
];
