import React from "react";
import { StyleSheet } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { PrivacySettings } from "../../api/types";
import { useMyProfile } from "../../hooks";
import { spacing } from "../../theme";
import { LoadingState, Screen, Section, SwitchRow, Text, showToast } from "../../ui";

export function PrivacyScreen() {
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const update = useMutation({
    mutationFn: (patch: Partial<PrivacySettings>) => profileApi.updateSettings(patch),
    onSuccess: (updated) => queryClient.setQueryData(queryKeys.profile, updated),
    onError: (e) => showToast(errorMessage(e), "error")
  });

  if (!profile.data)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  const settings = profile.data.settings;

  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="privacy-screen">
      <Section title="Visibilité" footer="En pause, tu restes visible pour tes matchs actuels et tu peux continuer à discuter.">
        <SwitchRow
          label="Apparaître dans Découvrir"
          testID="privacy-discoverable"
          description="Désactive pour mettre ton profil en pause."
          value={settings.discoverable}
          disabled={update.isPending}
          onValueChange={(discoverable) => update.mutate({ discoverable })}
        />
        <SwitchRow
          label="Afficher ma distance"
          testID="privacy-distance"
          description="Les autres voient « À 3 km », jamais ta position."
          value={settings.showDistance}
          disabled={update.isPending}
          onValueChange={(showDistance) => update.mutate({ showDistance })}
        />
      </Section>
      <Section title="Tes données">
        <Text tone="muted" style={styles.note}>
          Ton e-mail et ta date de naissance ne sont jamais affichés. Ta position est arrondie à environ 1 km. Les métadonnées de tes photos
          sont supprimées. Supprimer ton compte efface définitivement ton profil, tes photos, tes matchs et tes messages.
        </Text>
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  note: {
    padding: spacing.lg
  }
});
