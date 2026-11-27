import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { useMutation } from "@tanstack/react-query";
import { profileApi } from "../../../api/endpoints";
import { errorMessage } from "../../../api/client";
import { keys, queryClient } from "../../../hooks/queryClient";
import { useTheme } from "../../../theme/ThemeProvider";
import { spacing } from "../../../theme/palette";
import { Button, Card, Icon, InlineError, Text } from "../../../ui";

/**
 * Location is requested only on an explicit tap, with low accuracy. The
 * server snaps it to a ~1 km grid and only ever shows a rounded distance.
 */
export function LocationCard({ hasLocation }: { hasLocation: boolean }) {
  const { colors } = useTheme();
  const [permissionError, setPermissionError] = useState<string | null>(null);

  const update = useMutation({
    mutationFn: async () => {
      setPermissionError(null);
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        throw new Error("Permission refusée : tu peux l'activer plus tard dans les réglages de ton téléphone.");
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      return profileApi.setLocation(position.coords.latitude, position.coords.longitude);
    },
    onSuccess: (profile) => queryClient.setQueryData(keys.myProfile, profile),
    onError: (error) => setPermissionError(errorMessage(error))
  });
  const clear = useMutation({
    mutationFn: profileApi.clearLocation,
    onSuccess: (profile) => queryClient.setQueryData(keys.myProfile, profile)
  });

  return (
    <Card style={styles.card}>
      <View style={styles.row}>
        <Icon name={hasLocation ? "location" : "location-outline"} size={22} color={hasLocation ? colors.success : colors.textMuted} />
        <Text variant="bodyStrong" style={styles.flex}>
          {hasLocation ? "Position approximative enregistrée" : "Position non partagée"}
        </Text>
      </View>
      <Text variant="caption" tone="muted">
        Nous n’enregistrons qu’une zone d’environ 1 km et n’affichons qu’une distance arrondie. Ton adresse et tes coordonnées exactes ne sont
        jamais stockées ni montrées.
      </Text>
      <InlineError message={permissionError} />
      <View style={styles.actions}>
        <Button
          label={hasLocation ? "Mettre à jour" : "Utiliser ma position"}
          icon="navigate-outline"
          variant="secondary"
          size="sm"
          loading={update.isPending}
          onPress={() => update.mutate()}
          testID="location-update"
        />
        {hasLocation ? <Button label="Supprimer" variant="ghost" size="sm" loading={clear.isPending} onPress={() => clear.mutate()} /> : null}
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: { gap: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  flex: { flex: 1 },
  actions: { flexDirection: "row", gap: spacing.xs, flexWrap: "wrap" }
});
