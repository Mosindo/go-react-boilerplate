import React, { useState } from "react";
import * as Location from "expo-location";
import { View } from "react-native";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { Button, Input, Text, spacing } from "../../shared/ui";

const coarse = (v: number): number => Math.round(v * 100) / 100; // ~1 km: exact position never leaves the phone

type Props = { hasLocation: boolean; initialCity: string; onDone: () => void };

export function LocationStep({ hasLocation, initialCity, onDone }: Props) {
  const [city, setCity] = useState(initialCity);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const useGps = async () => {
    setBusy(true);
    setError(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setError("Permission refusée. Vous pouvez saisir votre ville ou passer cette étape.");
        return;
      }
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const lat = coarse(pos.coords.latitude);
      const lng = coarse(pos.coords.longitude);
      let label: string | undefined;
      try {
        const places = await Location.reverseGeocodeAsync({ latitude: lat, longitude: lng });
        label = places[0]?.city ?? places[0]?.subregion ?? undefined;
      } catch {
        label = undefined;
      }
      await profileApi.saveLocation(lat, lng, label ?? (city.trim() || undefined));
      onDone();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const saveCity = async () => {
    if (!city.trim()) {
      onDone();
      return;
    }
    setBusy(true);
    try {
      const current = await profileApi.me();
      await profileApi.save({ firstName: current.firstName, gender: current.gender, city: city.trim() });
      onDone();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <Text tone="muted">
        Votre position sert uniquement à trouver des profils proches. Elle est arrondie à environ 1 km avant d'être enregistrée,
        et les autres membres ne voient qu'une distance approximative, jamais votre adresse ni vos coordonnées.
      </Text>
      {hasLocation ? <Text tone="accent" variant="label">Position enregistrée</Text> : null}
      <Button title={hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"} onPress={useGps} loading={busy} testID="location-gps" />
      <Input label="Ou indiquez votre ville" value={city} onChangeText={setCity} maxLength={80} testID="location-city" hint="Sans position, la limite de distance n'est pas appliquée." />
      {error ? <Text tone="danger" accessibilityLiveRegion="polite">{error}</Text> : null}
      <Button title={city.trim() ? "Continuer avec cette ville" : "Passer cette étape"} variant="secondary" onPress={saveCity} disabled={busy} testID="location-skip" />
    </View>
  );
}
