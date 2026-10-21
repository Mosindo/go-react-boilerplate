import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { fetchPreferences, fetchProfile, saveProfile, savePreferences } from "../../api/profile";
import type { Preferences } from "../../api/types";
import { keys } from "../../hooks/queries";
import { getApproxLocation } from "../../lib/device";
import { draftFromProfile, draftToInput } from "../../lib/profileDraft";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, showError, showToast } from "../../shared/feedback";
import { ScreenContainer, Section } from "../../shared/layout";
import { Button, Card, Text } from "../../shared/ui";
import { spacing, useTheme } from "../../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Privacy">;

function SwitchRow({
  label,
  hint,
  value,
  onChange,
  testID
}: {
  label: string;
  hint: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID: string;
}) {
  const theme = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.flex}>
        <Text style={{ fontWeight: "600" }}>{label}</Text>
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      </View>
      <Switch
        testID={testID}
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        trackColor={{ true: theme.primary, false: theme.border }}
      />
    </View>
  );
}

export function PrivacyScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const prefs = useQuery({ queryKey: keys.preferences, queryFn: fetchPreferences });
  const profile = useQuery({ queryKey: keys.profile, queryFn: fetchProfile });

  const update = useMutation({
    mutationFn: (next: Preferences) => savePreferences(next),
    onSuccess: (next) => {
      queryClient.setQueryData(keys.preferences, next);
      void queryClient.invalidateQueries({ queryKey: keys.discover });
    },
    onError: (err) => showError(err instanceof Error ? err.message : "Modification impossible.")
  });

  const location = useMutation({
    mutationFn: async (mode: "refresh" | "clear") => {
      if (!profile.data) return;
      const draft = draftFromProfile(profile.data);
      if (mode === "clear") draft.clearLocation = true;
      else {
        const loc = await getApproxLocation();
        if (!loc) throw new Error("Position indisponible. Vérifiez l'autorisation de localisation.");
        draft.latitude = loc.latitude;
        draft.longitude = loc.longitude;
      }
      await saveProfile(draftToInput(draft));
    },
    onSuccess: async (_, mode) => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      showToast(mode === "clear" ? "Position supprimée." : "Position mise à jour.", "success");
    },
    onError: (err) => showError(err instanceof Error ? err.message : "Action impossible.")
  });

  if (prefs.isPending || profile.isPending)
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  if (prefs.isError || profile.isError || !prefs.data || !profile.data)
    return (
      <ScreenContainer>
        <ErrorView
          onRetry={() => {
            void prefs.refetch();
            void profile.refetch();
          }}
        />
      </ScreenContainer>
    );
  const p = prefs.data;
  const hasLocation = profile.data.latitude !== null;

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="privacy-screen">
      <Section title="Visibilité">
        <Card>
          <SwitchRow
            testID="switch-discoverable"
            label="Apparaître dans la découverte"
            hint="Désactivez pour mettre votre profil en pause. Vos matchs et conversations restent actifs."
            value={p.discoverable}
            onChange={(discoverable) => update.mutate({ ...p, discoverable })}
          />
          <SwitchRow
            testID="switch-show-distance"
            label="Afficher ma distance"
            hint="Les autres voient une distance approximative (jamais votre position)."
            value={p.showDistance}
            onChange={(showDistance) => update.mutate({ ...p, showDistance })}
          />
        </Card>
      </Section>

      <Section title="Position">
        <Card>
          <Text tone="muted">
            {hasLocation
              ? "Une position approximative (à ~1 km près) est enregistrée pour calculer les distances. Votre adresse et vos coordonnées exactes ne sont jamais conservées ni partagées."
              : "Aucune position enregistrée : les distances ne sont pas calculées."}
          </Text>
          <Button
            testID="location-refresh"
            label="Mettre à jour ma position"
            variant="secondary"
            loading={location.isPending}
            onPress={() => location.mutate("refresh")}
          />
          {hasLocation ? (
            <Button testID="location-clear" label="Supprimer ma position" variant="ghost" onPress={() => location.mutate("clear")} />
          ) : null}
        </Card>
      </Section>

      <Button label="Personnes bloquées" variant="secondary" onPress={() => navigation.navigate("BlockedUsers")} />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  flex: { flex: 1, gap: 2 }
});
