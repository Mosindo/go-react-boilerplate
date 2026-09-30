import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { Button, Text, TextField } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { profileApi } from "../../lib/api/endpoints";
import type { OwnProfile } from "../../lib/api/types";
import { showToast } from "../../lib/toast";
import { useProfileMutation } from "./hooks";

/**
 * Location is requested only when the member taps the button (foreground,
 * one-shot, low accuracy). Coordinates are rounded before leaving the device
 * and again on the server; only an approximate distance is ever shown.
 */
export function LocationControl({ profile }: { profile: OwnProfile }) {
  const [city, setCity] = useState(profile.city);
  const [locating, setLocating] = useState(false);
  const locationMutation = useProfileMutation((vars: { lat: number; lon: number; city?: string }) =>
    profileApi.updateLocation(vars.lat, vars.lon, vars.city)
  );
  const cityMutation = useProfileMutation((value: string) => profileApi.update({ city: value }));
  const clearMutation = useProfileMutation((_: void) => profileApi.clearLocation());

  const locate = async () => {
    setLocating(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        showToast("Sans localisation, vous verrez des profils sans filtre de distance.", "info");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      const lat = Math.round(position.coords.latitude * 100) / 100;
      const lon = Math.round(position.coords.longitude * 100) / 100;
      let detectedCity: string | undefined;
      try {
        const [place] = await Location.reverseGeocodeAsync({ latitude: lat, longitude: lon });
        detectedCity = place?.city ?? place?.subregion ?? undefined;
      } catch {
        // Reverse geocoding is optional (unavailable on some platforms).
      }
      const updated = await locationMutation.mutateAsync({ lat, lon, city: detectedCity ?? (city.trim() || undefined) });
      setCity(updated.city);
      showToast("Position approximative enregistrée", "success");
    } catch (e) {
      showToast(errorMessage(e, "Impossible d'obtenir votre position."), "error");
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={styles.wrap}>
      <Button
        icon="navigate"
        label={profile.hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
        loading={locating}
        onPress={locate}
        testID="location-use"
        variant={profile.hasLocation ? "secondary" : "primary"}
      />
      <Text tone="muted" variant="caption">
        {profile.hasLocation
          ? "Position approximative enregistrée (à ~1 km près). Les autres voient uniquement une distance arrondie."
          : "Utilisée uniquement pour calculer une distance approximative. Jamais affichée précisément."}
      </Text>
      <TextField
        label="Ville affichée (facultatif)"
        maxLength={80}
        onBlur={() => {
          if (city.trim() !== profile.city) cityMutation.mutate(city.trim());
        }}
        onChangeText={setCity}
        placeholder="Ex. Lyon"
        testID="location-city"
        value={city}
      />
      {profile.hasLocation ? (
        <Button label="Effacer ma position" onPress={() => clearMutation.mutate()} size="sm" variant="ghost" />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 10 }
});
