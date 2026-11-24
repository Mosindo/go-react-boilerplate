import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../queryClient";
import { colors, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { getApproximateLocation, LocationDeniedError } from "../../utils/location";
import { AppText } from "../AppText";
import { Button } from "../Button";
import { showToast } from "../Toast";

type Props = { hasLocation: boolean; city: string; onUpdated?: () => void };

export function LocationCard({ hasLocation, city, onUpdated }: Props) {
  const client = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = async () => {
    setBusy(true);
    setError(null);
    try {
      const location = await getApproximateLocation();
      const profile = await profileApi.setLocation(location.latitude, location.longitude, location.city);
      client.setQueryData(queryKeys.myProfile, profile);
      void client.resetQueries({ queryKey: queryKeys.discovery });
      showToast("Position mise à jour.", "success");
      onUpdated?.();
    } catch (e) {
      setError(
        e instanceof LocationDeniedError
          ? "Autorise l'accès à ta position dans les réglages pour découvrir des personnes près de toi."
          : errorMessage(e, "Impossible de récupérer ta position. Réessaie.")
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={styles.card}>
      <View style={styles.row}>
        <Ionicons name={hasLocation ? "location" : "location-outline"} size={22} color={hasLocation ? colors.success : colors.primary} />
        <AppText variant="bodyStrong" style={styles.flex}>
          {hasLocation ? `Position enregistrée${city ? ` · ${city}` : ""}` : "Position non définie"}
        </AppText>
      </View>
      <AppText variant="caption" color={colors.textMuted}>
        Nous arrondissons ta position à environ 1 km avant de l’enregistrer, et n’affichons qu’une distance approximative aux autres. Ta position n’est lue que quand tu appuies sur ce bouton.
      </AppText>
      {error ? (
        <AppText variant="caption" color={colors.danger}>
          {error}
        </AppText>
      ) : null}
      <Button label={hasLocation ? "Actualiser ma position" : "Utiliser ma position"} icon="navigate-outline" onPress={update} loading={busy} variant={hasLocation ? "secondary" : "primary"} testID="location-update" />
    </View>
  );
}

const styles = StyleSheet.create({
  card: { gap: spacing.sm, padding: spacing.md, borderRadius: radii.md, backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  flex: { flex: 1 }
});
