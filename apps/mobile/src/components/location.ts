import * as Location from "expo-location";
import { profileApi } from "../api/profile";
import type { MeProfile } from "../api/types";

export type LocationResult = { ok: true; profile: MeProfile } | { ok: false; reason: "denied" | "unavailable" };

/**
 * Asks for the foreground location (balanced accuracy is enough: the API
 * rounds coordinates to ~1 km) and sends it to the server.
 */
export async function shareLocation(prompt = true): Promise<LocationResult> {
  const permission = prompt ? await Location.requestForegroundPermissionsAsync() : await Location.getForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    return { ok: false, reason: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const profile = await profileApi.updateLocation(position.coords.latitude, position.coords.longitude);
    return { ok: true, profile };
  } catch {
    return { ok: false, reason: "unavailable" };
  }
}

const REFRESH_AFTER_MS = 12 * 60 * 60 * 1000;

/** Silently refreshes a stale location when permission was already granted. */
export async function refreshLocationIfStale(profile: MeProfile | undefined): Promise<MeProfile | null> {
  const updatedAt = profile?.profile?.locationUpdatedAt;
  if (!updatedAt || Date.now() - new Date(updatedAt).getTime() < REFRESH_AFTER_MS) {
    return null;
  }
  const result = await shareLocation(false);
  return result.ok ? result.profile : null;
}
