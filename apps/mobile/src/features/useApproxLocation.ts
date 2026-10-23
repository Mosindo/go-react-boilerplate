import { useCallback, useState } from "react";
import { Linking } from "react-native";
import * as Location from "expo-location";
import { roundCoordinate } from "../lib/format";

export type LocationState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "granted"; latitude: number; longitude: number }
  | { status: "denied"; canAskAgain: boolean }
  | { status: "error" };

/** Requests foreground permission and returns a coarse position (rounded, never precise). */
export function useApproxLocation(initial?: { latitude: number; longitude: number } | null) {
  const [state, setState] = useState<LocationState>(
    initial ? { status: "granted", ...initial } : { status: "idle" }
  );

  const request = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        setState({ status: "denied", canAskAgain: permission.canAskAgain });
        return;
      }
      const position = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced
      });
      setState({
        status: "granted",
        latitude: roundCoordinate(position.coords.latitude),
        longitude: roundCoordinate(position.coords.longitude)
      });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  const clear = useCallback(() => setState({ status: "idle" }), []);
  const openSettings = useCallback(() => {
    void Linking.openSettings();
  }, []);

  return { state, request, clear, openSettings };
}
