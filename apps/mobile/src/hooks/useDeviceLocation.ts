import { useCallback, useState } from "react";
import * as Location from "expo-location";
import { ApiError } from "../api/client";
import { profileApi } from "../api/platform";

export type LocationState = "idle" | "working" | "done" | "denied" | "error";

/**
 * Asks for foreground permission once, sends only a coarse position to the API
 * (the server rounds it to ~1 km and never returns it) and a city label.
 */
export function useDeviceLocation(onShared?: (city: string) => void) {
  const [state, setState] = useState<LocationState>("idle");
  const [message, setMessage] = useState<string | null>(null);

  const share = useCallback(async () => {
    setState("working");
    setMessage(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setState("denied");
        setMessage("Localisation refusée. Vous pouvez l'activer dans les réglages du téléphone.");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        city = place?.city ?? place?.subregion ?? place?.region ?? "";
      } catch {
        // reverse geocoding is a nicety; distance matching works without a label
      }
      await profileApi.setLocation(position.coords.latitude, position.coords.longitude, city);
      setState("done");
      setMessage(city ? `Position enregistrée près de ${city}.` : "Position enregistrée.");
      onShared?.(city);
    } catch (error) {
      setState("error");
      setMessage(error instanceof ApiError ? error.message : "Impossible de récupérer votre position.");
    }
  }, [onShared]);

  return { state, message, share };
}
