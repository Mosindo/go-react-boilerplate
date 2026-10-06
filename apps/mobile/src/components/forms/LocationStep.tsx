import React, { useState } from "react";
import { Linking, StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { Ionicons } from "@expo/vector-icons";
import { Button, Notice, Text, spacing, useTheme } from "../../shared/ui";

type LocationStepProps = {
  busy?: boolean;
  error?: string | null;
  currentCity?: string;
  submitLabel?: string;
  onLocated: (location: { latitude: number; longitude: number; city: string }) => void;
};

/**
 * Asks for foreground location once, resolves a city name on the device and
 * sends both to the server. The server rounds the coordinates to ~1 km and
 * never exposes them to other users.
 */
export function LocationStep({ busy = false, error, currentCity, submitLabel = "Utiliser ma position", onLocated }: LocationStepProps) {
  const { colors } = useTheme();
  const [working, setWorking] = useState(false);
  const [problem, setProblem] = useState<string | null>(null);
  const [denied, setDenied] = useState(false);

  const locate = async () => {
    setProblem(null);
    setWorking(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setDenied(true);
        setProblem("La localisation est désactivée pour l'application.");
        return;
      }
      setDenied(false);
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = "";
      try {
        const places = await Location.reverseGeocodeAsync(position.coords);
        city = places[0]?.city ?? places[0]?.subregion ?? places[0]?.region ?? "";
      } catch {
        // A city label is a nicety: continue without it.
      }
      onLocated({ latitude: position.coords.latitude, longitude: position.coords.longitude, city });
    } catch {
      setProblem("Impossible d'obtenir votre position. Vérifiez que la localisation est activée.");
    } finally {
      setWorking(false);
    }
  };

  return (
    <View style={styles.root}>
      <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name="location" size={32} />
      </View>
      {currentCity ? (
        <Text style={styles.center} tone="muted">
          Position actuelle : {currentCity}
        </Text>
      ) : null}
      <Text style={styles.center} tone="muted">
        Nous utilisons votre position pour vous proposer des personnes proches. Seule une position approximative (environ 1 km) est
        enregistrée. Votre adresse et vos coordonnées exactes ne sont jamais partagées : les autres voient seulement une distance arrondie.
      </Text>
      {problem || error ? <Notice message={(problem ?? error) as string} tone="danger" /> : null}
      <Button label={submitLabel} loading={busy || working} onPress={locate} testID="location-submit" />
      {problem ? (
        <Button
          label={denied ? "Ouvrir les réglages" : "Ouvrir les réglages de localisation"}
          onPress={() => void Linking.openSettings()}
          variant="ghost"
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.lg, alignItems: "center" },
  icon: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  center: { textAlign: "center" }
});
