import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useAuth } from "../../providers/AuthProvider";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, Card, Header, Screen, SectionTitle, showToast, Text } from "../../ui";
import { useMyProfile } from "./queries";
import { useLocationUpdate } from "./useLocationUpdate";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

function Row({ icon, label, onPress, testID }: { icon: keyof typeof Ionicons.glyphMap; label: string; onPress: () => void; testID?: string }) {
  const { colors, spacing } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.row, { paddingVertical: spacing.sm, opacity: pressed ? 0.7 : 1 }]}
      testID={testID}
    >
      <Ionicons color={colors.textMuted} name={icon} size={20} />
      <Text style={styles.flex}>{label}</Text>
      <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} />
    </Pressable>
  );
}

export function SettingsScreen({ navigation }: Props) {
  const { colors, spacing } = useTheme();
  const { user, signOut } = useAuth();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const location = useLocationUpdate();
  const onError = (e: unknown) => showToast(errorMessage(e), "error");

  const visibility = useMutation({
    mutationFn: profileApi.setDiscoverable,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.profile }),
    onError
  });
  const clearLocation = useMutation({
    mutationFn: profileApi.clearLocation,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      showToast("Position supprimée.", "success");
    },
    onError
  });

  const discoverable = profile.data?.discoverable ?? true;
  return (
    <Screen scroll header={<Header back title="Confidentialité et compte" />}>
      <SectionTitle>Visibilité</SectionTitle>
      <Card>
        <View style={styles.row}>
          <View style={styles.flex}>
            <Text variant="bodyStrong">Apparaître dans Découvrir</Text>
            <Text tone="muted" variant="caption">
              Désactive pour mettre ton profil en pause. Tes matchs et conversations restent accessibles.
            </Text>
          </View>
          <Switch
            accessibilityLabel="Apparaître dans Découvrir"
            onValueChange={(v) => visibility.mutate(v)}
            testID="settings-discoverable"
            thumbColor={colors.surface}
            trackColor={{ true: colors.primary, false: colors.border }}
            value={discoverable}
          />
        </View>
      </Card>

      <SectionTitle>Localisation</SectionTitle>
      <Card>
        <Text tone="muted" variant="caption">
          {profile.data?.hasLocation
            ? "Une position approximative (à ~1 km près) est enregistrée. Les autres ne voient qu’une distance arrondie, jamais ta position."
            : "Aucune position enregistrée : les distances ne sont pas calculées et tu vois des profils de partout."}
        </Text>
        <Button
          icon="navigate-outline"
          label="Mettre à jour ma position"
          loading={location.pending}
          onPress={() => void location.update()}
          size="sm"
          variant="secondary"
        />
        {profile.data?.hasLocation ? (
          <Button label="Supprimer ma position" loading={clearLocation.isPending} onPress={() => clearLocation.mutate()} size="sm" variant="ghost" />
        ) : null}
      </Card>

      <SectionTitle>Sécurité</SectionTitle>
      <Card style={{ paddingVertical: spacing.xs }}>
        <Row icon="ban-outline" label="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} />
        <Row icon="key-outline" label="Changer de mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
      </Card>

      <SectionTitle>Compte</SectionTitle>
      <Card>
        <Text tone="muted" variant="caption">
          Connecté·e avec {user?.email}
        </Text>
        <Button icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} testID="settings-logout" variant="secondary" />
        <Button label="Supprimer mon compte" onPress={() => navigation.navigate("DeleteAccount")} testID="settings-delete" variant="danger" />
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 12 },
  flex: { flex: 1 }
});
