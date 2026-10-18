import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { changePassword, deleteAccount } from "../api/auth";
import { errorMessage } from "../api/client";
import { updateSettings } from "../api/profile";
import { ConfirmModal } from "../components/ConfirmModal";
import { NavHeader } from "../components/NavHeader";
import { Screen } from "../components/Screen";
import { APP_NAME } from "../components/labels";
import { queryKeys, useMyProfile } from "../hooks/queries";
import { useAuth, useCurrentUser } from "../hooks/useAuth";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Button, Input, Notice, Text, colors, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "Settings">;

export default function SettingsScreen({ navigation }: Props) {
  const client = useQueryClient();
  const user = useCurrentUser();
  const { signOut, forgetSession } = useAuth();
  const { data: profile } = useMyProfile();
  const [dialog, setDialog] = useState<"password" | "delete" | "logout" | null>(null);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [deletePassword, setDeletePassword] = useState("");

  const settings = useMutation({
    mutationFn: updateSettings,
    onSuccess: (saved) => {
      client.setQueryData(queryKeys.profile, saved);
      void client.invalidateQueries({ queryKey: queryKeys.discover });
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  const password = useMutation({
    mutationFn: () => changePassword(currentPassword, newPassword),
    onSuccess: () => {
      showToast("Mot de passe modifié.", "success");
      setDialog(null);
      setCurrentPassword("");
      setNewPassword("");
    }
  });

  const remove = useMutation({
    mutationFn: () => deleteAccount(deletePassword),
    onSuccess: async () => {
      await forgetSession();
      showToast("Votre compte et vos données ont été supprimés.", "info");
    }
  });

  return (
    <Screen
      header={<NavHeader onBack={() => navigation.goBack()} title="Paramètres" />}
      scroll
      testID="settings-screen"
    >
      <Text tone="muted">Connecté avec {user.email}</Text>

      <View style={styles.group}>
        <Text variant="heading" weight="bold">
          Confidentialité
        </Text>
        <ToggleRow
          description="Désactivez pour mettre votre profil en pause : vous ne serez plus proposé(e), mais vos matchs et conversations restent actifs."
          label="Afficher mon profil dans la découverte"
          onChange={(value) => settings.mutate({ isVisible: value })}
          testID="toggle-visible"
          value={profile?.isVisible ?? true}
        />
        <ToggleRow
          description="Les autres voient une distance approximative, jamais votre position exacte."
          label="Afficher ma distance"
          onChange={(value) => settings.mutate({ showDistance: value })}
          testID="toggle-distance"
          value={profile?.showDistance ?? true}
        />
        <Button
          label="Personnes bloquées"
          onPress={() => navigation.navigate("BlockedUsers")}
          testID="open-blocked"
          variant="outline"
        />
      </View>

      <View style={styles.group}>
        <Text variant="heading" weight="bold">
          Compte
        </Text>
        <Button
          label="Changer de mot de passe"
          onPress={() => setDialog("password")}
          testID="open-change-password"
          variant="outline"
        />
        <Button label="Se déconnecter" onPress={() => setDialog("logout")} testID="logout" variant="outline" />
        <Button
          label="Supprimer mon compte"
          onPress={() => setDialog("delete")}
          testID="open-delete-account"
          variant="destructive"
        />
      </View>

      <Notice
        description="Pas d'abonnement, pas de limite de likes, pas de boost payant, pas de publicité obligatoire. Toutes les fonctionnalités sont accessibles à tout le monde."
        title={`${APP_NAME} est 100 % gratuit`}
        tone="success"
      />

      <ConfirmModal
        confirmLabel="Se déconnecter"
        onCancel={() => setDialog(null)}
        onConfirm={() => void signOut()}
        title="Se déconnecter ?"
        visible={dialog === "logout"}
      />

      <ConfirmModal
        confirmLabel="Modifier"
        loading={password.isPending}
        onCancel={() => setDialog(null)}
        onConfirm={() => password.mutate()}
        title="Changer de mot de passe"
        visible={dialog === "password"}
      >
        {password.isError ? <Notice title={errorMessage(password.error)} tone="danger" /> : null}
        <Input
          autoCapitalize="none"
          label="Mot de passe actuel"
          onChangeText={setCurrentPassword}
          secureTextEntry
          testID="current-password"
          value={currentPassword}
        />
        <Input
          autoCapitalize="none"
          helperText="8 caractères minimum."
          label="Nouveau mot de passe"
          onChangeText={setNewPassword}
          secureTextEntry
          testID="new-password"
          value={newPassword}
        />
      </ConfirmModal>

      <ConfirmModal
        confirmLabel="Supprimer définitivement"
        destructive
        loading={remove.isPending}
        message="Votre profil, vos photos, vos matchs et vos messages seront supprimés définitivement. Cette action est irréversible."
        onCancel={() => setDialog(null)}
        onConfirm={() => remove.mutate()}
        title="Supprimer mon compte ?"
        visible={dialog === "delete"}
      >
        {remove.isError ? <Notice title={errorMessage(remove.error)} tone="danger" /> : null}
        <Input
          autoCapitalize="none"
          label="Confirmez avec votre mot de passe"
          onChangeText={setDeletePassword}
          secureTextEntry
          testID="delete-password"
          value={deletePassword}
        />
      </ConfirmModal>
    </Screen>
  );
}

type ToggleRowProps = {
  label: string;
  description: string;
  value: boolean;
  onChange: (value: boolean) => void;
  testID: string;
};

function ToggleRow({ label, description, value, onChange, testID }: ToggleRowProps) {
  return (
    <View style={styles.toggle}>
      <View style={styles.toggleCopy}>
        <Text weight="semibold">{label}</Text>
        <Text tone="muted" variant="caption">
          {description}
        </Text>
      </View>
      <Switch
        accessibilityLabel={label}
        onValueChange={onChange}
        thumbColor={colors.inverse}
        trackColor={{ false: colors.borderStrong, true: colors.primary }}
        testID={testID}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.md },
  toggle: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  toggleCopy: { flex: 1, gap: 2 }
});
