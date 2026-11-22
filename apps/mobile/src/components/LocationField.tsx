import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { updateLocation } from "../api/dating";
import { useProfileMutation } from "../hooks/queries";
import { getApproximatePosition } from "../lib/location";
import { Button, Text, TextField, colors, radii, showToast, spacing } from "../ui";

type Props = {
  hasLocation: boolean;
  city: string;
  onSaved?: () => void;
};

/** Requests the device position once and stores a rounded, approximate location. */
export function LocationField({ hasLocation, city, onSaved }: Props) {
  const [cityLabel, setCityLabel] = useState(city);
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const mutation = useProfileMutation((p: { latitude: number; longitude: number; city: string }) =>
    updateLocation(p.latitude, p.longitude, p.city)
  );

  const locate = async () => {
    setError(null);
    setLocating(true);
    try {
      const pos = await getApproximatePosition();
      const label = (cityLabel.trim() || pos.city).slice(0, 80);
      setCityLabel(label);
      await mutation.mutateAsync({ latitude: pos.latitude, longitude: pos.longitude, city: label });
      showToast("Position mise à jour", "success");
      onSaved?.();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={styles.wrap}>
      <View style={styles.status}>
        <Text variant="label">{hasLocation ? "Position enregistrée" : "Aucune position"}</Text>
        <Text tone="muted" variant="caption">
          Nous ne conservons qu’une position arrondie (≈ 1 km). Les autres membres ne voient qu’une distance approximative, jamais votre
          adresse.
        </Text>
      </View>
      <TextField
        label="Ville affichée (facultatif)"
        maxLength={80}
        onChangeText={setCityLabel}
        placeholder="Ex. Lyon"
        testID="location-city"
        value={cityLabel}
      />
      <Button
        icon="navigate-outline"
        label={hasLocation ? "Actualiser ma position" : "Utiliser ma position"}
        loading={locating || mutation.isPending}
        onPress={locate}
        testID="location-use"
        variant={hasLocation ? "secondary" : "primary"}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.md
  },
  status: {
    gap: 4,
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  }
});
