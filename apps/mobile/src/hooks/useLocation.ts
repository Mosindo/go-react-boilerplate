import { useCallback, useEffect } from "react";
import { Platform } from "react-native";
import * as Location from "expo-location";
import { profileApi } from "../api/endpoints";

export type LocationOutcome = "updated" | "denied" | "unavailable";

/**
 * Requests foreground location only when needed, sends a single position to
 * the API (which coarsens it to ~1 km) and never stores it on the device.
 */
export function useLocationSync() {
  return useCallback(async (): Promise<LocationOutcome> => {
    try {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== "granted") return "denied";
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      let city: string | undefined;
      if (Platform.OS !== "web") {
        try {
          const [place] = await Location.reverseGeocodeAsync({ latitude: pos.coords.latitude, longitude: pos.coords.longitude });
          city = place?.city ?? place?.subregion ?? undefined;
        } catch {
          /* reverse geocoding is optional */
        }
      }
      await profileApi.updateLocation(pos.coords.latitude, pos.coords.longitude, city);
      return "updated";
    } catch {
      return "unavailable";
    }
  }, []);
}

const REFRESH_EVERY_MS = 6 * 3600_000;
let lastRefresh = 0;

/**
 * Silently refreshes the coarse location at most every 6 hours while the app
 * is used, only if the permission was already granted (never prompts).
 */
export function useLocationRefresh(enabled: boolean, onUpdated: () => void) {
  useEffect(() => {
    if (!enabled || Date.now() - lastRefresh < REFRESH_EVERY_MS) return;
    lastRefresh = Date.now();
    (async () => {
      try {
        const { status } = await Location.getForegroundPermissionsAsync();
        if (status !== "granted") return;
        const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
        await profileApi.updateLocation(pos.coords.latitude, pos.coords.longitude);
        onUpdated();
      } catch {
        /* best effort */
      }
    })();
  }, [enabled, onUpdated]);
}
