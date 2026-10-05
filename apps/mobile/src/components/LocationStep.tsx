import * as Location from "expo-location";
import { useState } from "react";
import { saveLocation } from "../api/platform";
import { errorMessage } from "../lib/errors";
import { Button, Notice, Text } from "../shared/ui";

/** Asks for the position once, sends only coarse coordinates and a city name to the API. */
export function LocationStep({ onSaved, label = "Utiliser ma position" }: { onSaved: () => void; label?: string }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const locate = async () => {
    setLoading(true);
    setError(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setError("Sans accès à votre position, nous ne pouvons pas vous proposer de personnes proches. Autorisez la localisation dans les réglages puis réessayez.");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        city = place?.city ?? place?.subregion ?? place?.region ?? "";
      } catch {
        // The city name is optional (reverse geocoding is unavailable on some platforms).
      }
      await saveLocation(position.coords.latitude, position.coords.longitude, city);
      onSaved();
    } catch (e) {
      setError(errorMessage(e, "Impossible de récupérer votre position. Réessayez."));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Text tone="muted">
        Votre position exacte n'est jamais affichée ni partagée : les autres voient seulement une distance approximative (par tranches de 5 km). Nous n'en conservons qu'une version arrondie à environ 1 km.
      </Text>
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="location-use" label={label} icon="location-outline" onPress={locate} loading={loading} />
    </>
  );
}
