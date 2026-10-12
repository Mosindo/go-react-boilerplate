import * as Location from "expo-location";
import { useCallback, useState } from "react";
import { datingApi } from "../api/dating";
import type { MyProfile } from "../api/types";

export type LocationResult = { ok: true; profile: MyProfile } | { ok: false; reason: "denied" | "unavailable" };

/**
 * Asks for foreground location only when the user taps the button, then sends a
 * single position to the API (which stores it rounded to ~1 km). Nothing is
 * tracked in the background.
 */
export function useLocationUpdate() {
  const [loading, setLoading] = useState(false);

  const update = useCallback(async (): Promise<LocationResult> => {
    setLoading(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        return { ok: false, reason: "denied" };
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city: string | undefined;
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        city = place?.city ?? place?.subregion ?? undefined;
      } catch {
        city = undefined;
      }
      const profile = await datingApi.saveLocation(position.coords.latitude, position.coords.longitude, city);
      return { ok: true, profile };
    } catch {
      return { ok: false, reason: "unavailable" };
    } finally {
      setLoading(false);
    }
  }, []);

  return { update, loading };
}
