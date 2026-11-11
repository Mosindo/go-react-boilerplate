import React from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi, safetyApi } from "../../api/endpoints";
import { LocationControl } from "../../features/LocationControl";
import { keys, useBlocks, useMyProfile, useProfileMutation } from "../../hooks/queries";
import { Button, LoadingState, Row, Screen, Section, Text, Toggle, useFeedback } from "../../ui";

export function PrivacyScreen() {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const { data } = useMyProfile();
  const blocks = useBlocks();
  const visibility = useProfileMutation(profileApi.setVisibility);
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => void qc.invalidateQueries({ queryKey: keys.blocks }),
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (!data) {
    return <LoadingState />;
  }

  const toggle = (field: "discoverable" | "showDistance", value: boolean) =>
    visibility.mutate({ [field]: value }, { onError: (e) => toast(errorMessage(e), "error") });

  return (
    <Screen edges={[]} testID="privacy-screen">
      <Section title="Visibilité">
        <Row
          icon="eye-outline"
          label="Apparaître dans la découverte"
          right={
            <Toggle
              testID="privacy-discoverable"
              accessibilityLabel="Apparaître dans la découverte"
              value={data.discoverable}
              onValueChange={(v) => toggle("discoverable", v)}
            />
          }
        />
        <Row
          icon="navigate-circle-outline"
          label="Afficher ma distance"
          last
          right={
            <Toggle
              accessibilityLabel="Afficher ma distance"
              value={data.showDistance}
              onValueChange={(v) => toggle("showDistance", v)}
            />
          }
        />
      </Section>
      <Text variant="caption" tone="subtle">
        Masquer ton profil le retire de la découverte. Tes matchs actuels peuvent toujours t’écrire.
      </Text>
      <Section title="Localisation">
        <View style={{ padding: 16 }}>
          <LocationControl hasLocation={data.hasLocation} onDone={() => toast("Position mise à jour", "success")} />
        </View>
      </Section>
      <Section title="Profils bloqués">
        {blocks.isLoading ? (
          <LoadingState />
        ) : (blocks.data ?? []).length === 0 ? (
          <View style={{ padding: 16 }}>
            <Text tone="muted">Tu n’as bloqué personne.</Text>
          </View>
        ) : (
          (blocks.data ?? []).map((b, i, list) => (
            <Row
              key={b.userId}
              icon="ban-outline"
              label={b.firstName || "Membre"}
              last={i === list.length - 1}
              right={<Button label="Débloquer" variant="ghost" size="md" onPress={() => unblock.mutate(b.userId)} />}
            />
          ))
        )}
      </Section>
    </Screen>
  );
}
