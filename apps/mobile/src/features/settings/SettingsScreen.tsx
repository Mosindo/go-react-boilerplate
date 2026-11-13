import { useMutation } from "@tanstack/react-query";
import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { authApi } from "../../api/auth";
import type { RootScreenProps } from "../../navigation/types";
import { useMyProfile, useUpdateProfile } from "../../state/queries";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Divider, Header, ListRow, Screen, Sheet, SwitchRow, Text, TextField, useToast } from "../../ui";
import { describeError } from "../../utils/errors";

export default function SettingsScreen({ navigation }: RootScreenProps<"Settings">) {
  const { colors, spacing, radii } = useTheme();
  const toast = useToast();
  const { signOut, forgetSession } = useSession();
  const { data } = useMyProfile();
  const update = useUpdateProfile();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [password, setPassword] = useState("");

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: () => {
      setConfirmDelete(false);
      void forgetSession();
    }
  });

  const toggle = (field: "isPaused" | "showDistance", value: boolean) =>
    update.mutate({ [field]: value }, { onError: (e) => toast.show(describeError(e), "error") });

  const group = [styles.group, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }];

  return (
    <Screen scroll testID="settings-screen">
      <Header back title="Confidentialité et sécurité" />
      <View style={{ gap: spacing.xl }}>
        <View style={{ gap: spacing.sm }}>
          <Text variant="label" weight="semibold" tone="muted">
            VISIBILITÉ
          </Text>
          <View style={group}>
            <SwitchRow
              label="Mettre mon profil en pause"
              description="Vous n'apparaissez plus dans la découverte. Vos matchs et messages restent accessibles."
              value={data?.isPaused ?? false}
              onValueChange={(v) => toggle("isPaused", v)}
              testID="settings-pause"
            />
            <Divider />
            <SwitchRow
              label="Afficher ma distance"
              description="Si désactivé, seule votre ville est visible."
              value={data?.showDistance ?? true}
              onValueChange={(v) => toggle("showDistance", v)}
              testID="settings-distance"
            />
          </View>
          <Text variant="caption" tone="subtle">
            Votre position exacte, votre e-mail et votre date de naissance ne sont jamais montrés aux autres membres.
          </Text>
        </View>

        <View style={{ gap: spacing.sm }}>
          <Text variant="label" weight="semibold" tone="muted">
            SÉCURITÉ
          </Text>
          <View style={group}>
            <ListRow
              icon="ban-outline"
              label="Membres bloqués"
              onPress={() => navigation.navigate("BlockedUsers")}
              testID="settings-blocked"
            />
          </View>
          <Text variant="caption" tone="subtle">
            Un membre bloqué ne peut plus vous voir ni vous écrire. Un signalement bloque aussi automatiquement la
            personne.
          </Text>
        </View>

        <View style={{ gap: spacing.sm }}>
          <Text variant="label" weight="semibold" tone="muted">
            COMPTE
          </Text>
          <View style={group}>
            <ListRow
              icon="log-out-outline"
              label="Se déconnecter"
              onPress={() => void signOut()}
              testID="settings-logout"
            />
            <Divider />
            <ListRow
              icon="trash-outline"
              label="Supprimer mon compte"
              description="Supprime définitivement profil, photos, matchs et messages."
              destructive
              onPress={() => setConfirmDelete(true)}
              testID="settings-delete"
            />
          </View>
        </View>
      </View>

      <Sheet
        visible={confirmDelete}
        title="Supprimer définitivement votre compte ?"
        onClose={() => setConfirmDelete(false)}
      >
        <View style={{ gap: spacing.md }}>
          <Text tone="muted">
            Cette action est irréversible. Votre profil, vos photos, vos matchs et vos messages seront supprimés.
          </Text>
          <TextField
            label="Mot de passe"
            value={password}
            onChangeText={setPassword}
            secureToggle
            testID="delete-password"
          />
          {remove.error ? <Text tone="danger">{describeError(remove.error)}</Text> : null}
          <Button
            label="Supprimer mon compte"
            variant="danger"
            disabled={!password}
            loading={remove.isPending}
            onPress={() => remove.mutate()}
            testID="delete-confirm"
          />
        </View>
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  group: { paddingHorizontal: 14, borderWidth: StyleSheet.hairlineWidth }
});
