import { useCallback, useState } from "react";
import { Platform } from "react-native";
import * as Location from "expo-location";

export type DeviceLocation = { latitude: number; longitude: number; city: string };

type LocationState = { loading: boolean; error: string | null };

/**
 * One-shot position lookup, only triggered by an explicit user action. Coordinates are sent to the
 * server, which snaps them to a ~1 km grid; the device never streams its position.
 */
export function useDeviceLocation() {
  const [state, setState] = useState<LocationState>({ loading: false, error: null });

  const locate = useCallback(async (): Promise<DeviceLocation | null> => {
    setState({ loading: true, error: null });
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setState({ loading: false, error: "Autorisez la localisation dans les réglages, ou saisissez votre ville." });
        return null;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      if (Platform.OS !== "web") {
        try {
          const [place] = await Location.reverseGeocodeAsync(position.coords);
          city = place?.city ?? place?.subregion ?? place?.region ?? "";
        } catch {
          // Reverse geocoding is a convenience; the user can type the city.
        }
      }
      setState({ loading: false, error: null });
      return { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
    } catch {
      setState({ loading: false, error: "Position indisponible. Réessayez ou saisissez votre ville." });
      return null;
    }
  }, []);

  return { ...state, locate };
}
