import { useCallback, useState } from "react";
import * as Location from "expo-location";

export type Located = { latitude: number; longitude: number; city: string };

/** Device GPS / geocoder access. Coordinates are rounded again server-side. */
export function useDeviceLocation() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const locate = useCallback(async (): Promise<Located | null> => {
    setBusy(true);
    setError(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setError("Autorisation refusée. Vous pouvez saisir une ville à la place.");
        return null;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        city = place?.city ?? place?.subregion ?? place?.region ?? "";
      } catch {
        // city label is optional
      }
      return { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
    } catch {
      setError("Position indisponible. Saisissez une ville à la place.");
      return null;
    } finally {
      setBusy(false);
    }
  }, []);

  const geocode = useCallback(async (query: string): Promise<Located | null> => {
    setBusy(true);
    setError(null);
    try {
      const results = await Location.geocodeAsync(query.trim());
      const first = results[0];
      if (!first) {
        setError("Ville introuvable.");
        return null;
      }
      return { latitude: first.latitude, longitude: first.longitude, city: query.trim() };
    } catch {
      setError("Recherche de ville indisponible sur cet appareil. Utilisez votre position.");
      return null;
    } finally {
      setBusy(false);
    }
  }, []);

  return { locate, geocode, busy, error };
}
