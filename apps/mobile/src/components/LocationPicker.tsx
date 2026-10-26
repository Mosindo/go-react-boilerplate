import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api";
import { geocodeCity, getCurrentPosition, type Position } from "../lib/location";
import { queryKeys } from "../lib/queryKeys";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { useFeedback } from "./Feedback";
import { Text } from "./Text";
import { TextField } from "./TextField";

type Props = { initialCity?: string; hasLocation: boolean; onSaved?: () => void };

/** Sets the approximate position from GPS or from a typed city. */
export function LocationPicker({ initialCity = "", hasLocation, onSaved }: Props) {
  const queryClient = useQueryClient();
  const theme = useTheme();
  const { toast } = useFeedback();
  const [city, setCity] = useState(initialCity);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: async (source: "gps" | "city") => {
      let position: Position;
      if (source === "gps") {
        position = await getCurrentPosition();
        if (city.trim()) position = { ...position, city: city.trim() };
      } else {
        if (!city.trim()) throw new Error("Saisissez le nom de votre ville.");
        position = await geocodeCity(city.trim());
      }
      return profileApi.saveLocation(position.latitude, position.longitude, position.city);
    },
    onSuccess: (profile) => {
      setError(null);
      setCity(profile.city);
      void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      if (onSaved) onSaved();
      else toast("Position enregistrée", "success");
    },
    onError: (err) => setError(err instanceof Error ? err.message : "Position indisponible.")
  });

  return (
    <View style={styles.wrap}>
      <Text muted>
        Nous n&apos;enregistrons qu&apos;une position approximative (environ 1 km). Les autres ne voient
        jamais votre adresse ni vos coordonnées, seulement une distance arrondie.
      </Text>
      <TextField
        label="Ville"
        value={city}
        onChangeText={setCity}
        placeholder="Paris"
        autoCapitalize="words"
        maxLength={80}
      />
      <Button
        label={hasLocation ? "Actualiser avec ma position" : "Utiliser ma position"}
        onPress={() => save.mutate("gps")}
        loading={save.isPending && save.variables === "gps"}
        testID="use-gps"
      />
      <Button
        label="Utiliser la ville saisie"
        variant="secondary"
        onPress={() => save.mutate("city")}
        loading={save.isPending && save.variables === "city"}
      />
      {error ? (
        <Text variant="caption" color={theme.danger} accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.md } });
