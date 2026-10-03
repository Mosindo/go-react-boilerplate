import { useCallback, useState } from "react";
import * as Location from "expo-location";

export type DeviceLocation = { latitude: number; longitude: number; city: string };

type State = { status: "idle" | "loading" | "denied" | "error" | "ready"; location: DeviceLocation | null };

/**
 * One-shot, foreground-only location lookup with balanced (city-level) accuracy.
 * The server rounds coordinates to ~1 km before storing them.
 */
export function useDeviceLocation() {
  const [state, setState] = useState<State>({ status: "idle", location: null });

  const request = useCallback(async (): Promise<DeviceLocation | null> => {
    setState((s) => ({ ...s, status: "loading" }));
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setState({ status: "denied", location: null });
        return null;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      try {
        const places = await Location.reverseGeocodeAsync(position.coords);
        city = places[0]?.city ?? places[0]?.subregion ?? "";
      } catch {
        // The city label is optional.
      }
      const location = { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
      setState({ status: "ready", location });
      return location;
    } catch {
      setState({ status: "error", location: null });
      return null;
    }
  }, []);

  return { ...state, request };
}
