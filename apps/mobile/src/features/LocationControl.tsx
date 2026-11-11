import React, { useState } from "react";
import { View } from "react-native";
import { errorMessage } from "../api/client";
import { profileApi } from "../api/endpoints";
import { useProfileMutation } from "../hooks/queries";
import { Button, Text } from "../ui";
import { getApproximatePosition } from "./media";

type Props = { hasLocation: boolean; onDone?: () => void };

export function LocationControl({ hasLocation, onDone }: Props) {
  const [error, setError] = useState<string | null>(null);
  const save = useProfileMutation(({ latitude, longitude }: { latitude: number; longitude: number }) =>
    profileApi.setLocation(latitude, longitude)
  );
  const clear = useProfileMutation<void>(() => profileApi.clearLocation());
  const [locating, setLocating] = useState(false);

  const locate = async () => {
    setError(null);
    setLocating(true);
    try {
      const pos = await getApproximatePosition();
      await save.mutateAsync(pos);
      onDone?.();
    } catch (e) {
      setError(errorMessage(e, e instanceof Error ? e.message : "Localisation impossible."));
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={{ gap: 12 }}>
      <Text tone="muted">
        Ta position sert uniquement à calculer une distance approximative (arrondie au kilomètre). Elle n’est jamais
        affichée, ni suivie en arrière-plan.
      </Text>
      <Button
        testID="location-use"
        label={hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
        icon="navigate-outline"
        loading={locating}
        onPress={locate}
      />
      {hasLocation ? (
        <Button
          label="Supprimer ma position"
          variant="ghost"
          size="md"
          loading={clear.isPending}
          onPress={() => clear.mutate()}
        />
      ) : null}
      {error ? <Text tone="danger">{error}</Text> : null}
    </View>
  );
}
