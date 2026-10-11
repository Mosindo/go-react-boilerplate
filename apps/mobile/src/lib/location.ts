import * as Location from 'expo-location';

export type LocationResult =
  | { ok: true; latitude: number; longitude: number }
  | { ok: false; reason: 'denied' | 'unavailable' };

/** ~1 km precision: the app never needs (or sends) anything finer. */
const round = (n: number): number => Math.round(n * 100) / 100;

/** Foreground-only permission; the result is only ever sent to PUT /me/location. */
export async function getApproxLocation(): Promise<LocationResult> {
  try {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (permission.status !== 'granted') return { ok: false, reason: 'denied' };
    const position = await Location.getCurrentPositionAsync({
      accuracy: Location.Accuracy.Low,
    });
    return {
      ok: true,
      latitude: round(position.coords.latitude),
      longitude: round(position.coords.longitude),
    };
  } catch {
    return { ok: false, reason: 'unavailable' };
  }
}
