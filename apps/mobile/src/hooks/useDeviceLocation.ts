import { useCallback, useState } from "react";
import { Linking, Platform } from "react-native";
import * as Location from "expo-location";

export type DeviceLocationStatus = "idle" | "requesting" | "granted" | "denied" | "unavailable";

export type Coordinates = { latitude: number; longitude: number };

/**
 * Foreground, coarse location only (Accuracy.Low). The server rounds the coordinates again before storing them.
 */
export function useDeviceLocation() {
  const [status, setStatus] = useState<DeviceLocationStatus>("idle");
  const [canAskAgain, setCanAskAgain] = useState(true);

  const request = useCallback(async (): Promise<Coordinates | null> => {
    setStatus("requesting");
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      setCanAskAgain(permission.canAskAgain);
      if (permission.status !== Location.PermissionStatus.GRANTED) {
        setStatus("denied");
        return null;
      }
      const enabled = await Location.hasServicesEnabledAsync();
      if (!enabled) {
        setStatus("unavailable");
        return null;
      }
      const position =
        (await Location.getLastKnownPositionAsync()) ??
        (await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low }));
      setStatus("granted");
      return { latitude: position.coords.latitude, longitude: position.coords.longitude };
    } catch {
      setStatus("unavailable");
      return null;
    }
  }, []);

  const openSettings = useCallback(async () => {
    try {
      if (Platform.OS === "web") {
        return;
      }
      await Linking.openSettings();
    } catch {
      // Nothing else we can do; the on-screen instructions still apply.
    }
  }, []);

  return { status, canAskAgain, request, openSettings };
}
