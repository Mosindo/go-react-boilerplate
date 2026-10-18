import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";

export type Coordinates = { latitude: number; longitude: number };

type LocationPickerProps = {
  coordinates: Coordinates | null;
  city: string;
  onChange: (coordinates: Coordinates, city?: string) => void;
  onCityChange: (city: string) => void;
};

/** Fallback when location permission is refused: an approximate area is enough. */
const CITIES: { name: string; latitude: number; longitude: number }[] = [
  { name: "Paris", latitude: 48.8566, longitude: 2.3522 },
  { name: "Lyon", latitude: 45.764, longitude: 4.8357 },
  { name: "Marseille", latitude: 43.2965, longitude: 5.3698 },
  { name: "Toulouse", latitude: 43.6047, longitude: 1.4442 },
  { name: "Bordeaux", latitude: 44.8378, longitude: -0.5792 },
  { name: "Lille", latitude: 50.6292, longitude: 3.0573 },
  { name: "Nantes", latitude: 47.2184, longitude: -1.5536 },
  { name: "Strasbourg", latitude: 48.5734, longitude: 7.7521 },
  { name: "Nice", latitude: 43.7102, longitude: 7.262 },
  { name: "Bruxelles", latitude: 50.8503, longitude: 4.3517 },
  { name: "Genève", latitude: 46.2044, longitude: 6.1432 },
  { name: "Montréal", latitude: 45.5017, longitude: -73.5673 }
];

export function LocationPicker({ coordinates, city, onChange, onCityChange }: LocationPickerProps) {
  const [loading, setLoading] = useState(false);
  const [problem, setProblem] = useState<string | null>(null);

  const locate = async () => {
    setLoading(true);
    setProblem(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setProblem("Localisation refusée. Choisissez une ville ci-dessous.");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let detectedCity: string | undefined;
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        detectedCity = place?.city ?? place?.subregion ?? undefined;
      } catch {
        // Reverse geocoding is unavailable on some platforms: the city stays editable.
      }
      onChange({ latitude: position.coords.latitude, longitude: position.coords.longitude }, detectedCity);
    } catch {
      setProblem("Position introuvable. Choisissez une ville ci-dessous.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.wrap}>
      <Text tone="muted">
        Votre position précise n'est jamais affichée : les autres voient seulement une distance approximative, et nous
        n'enregistrons qu'une position arrondie à environ 1 km.
      </Text>
      <Button
        label={coordinates ? "Mettre à jour ma position" : "Utiliser ma position"}
        loading={loading}
        onPress={locate}
        testID="locate-me"
        variant="secondary"
      />
      {problem ? <Notice title={problem} tone="warning" /> : null}
      {coordinates ? (
        <Notice title="Position enregistrée." tone="success" />
      ) : (
        <Text tone="muted" variant="caption">
          Ou choisissez une zone approximative :
        </Text>
      )}
      <ChipRow>
        {CITIES.map((c) => (
          <Chip
            key={c.name}
            label={c.name}
            onPress={() => {
              onChange({ latitude: c.latitude, longitude: c.longitude }, c.name);
            }}
            selected={city === c.name && coordinates?.latitude === c.latitude}
            testID={`city-${c.name}`}
          />
        ))}
      </ChipRow>
      <Input
        label="Ville affichée (facultatif)"
        maxLength={80}
        onChangeText={onCityChange}
        placeholder="Ex. Lyon"
        testID="city-input"
        value={city}
      />
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.md } });
