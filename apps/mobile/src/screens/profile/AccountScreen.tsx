import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useSession } from "../../hooks/useSession";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Dialog, Header, ListRow, Screen, Sheet, Text, TextField, showToast } from "../../ui";
import { validatePassword } from "../../utils/validation";

export function AccountScreen({ navigation }: AppScreenProps<"Account">) {
  const { user, signOut, forgetSession } = useSession();
  const [changing, setChanging] = useState(false);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [confirmLogout, setConfirmLogout] = useState(false);

  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      setChanging(false);
      setCurrent("");
      setNext("");
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
    }
  });

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(deletePassword),
    onSuccess: async () => {
      setDeleting(false);
      await forgetSession();
      showToast("Votre compte et vos données ont été supprimés.", "success");
    }
  });

  const nextError = next ? validatePassword(next) : null;

  return (
    <Screen scroll testID="account-screen">
      <Header title="Compte" onBack={navigation.goBack} />
      <Text variant="small" color="textMuted" style={styles.email}>
        Connecté avec {user?.email}
      </Text>
      <View style={styles.group}>
        <ListRow icon="key-outline" title="Changer de mot de passe" onPress={() => setChanging(true)} />
        <ListRow icon="log-out-outline" title="Se déconnecter" onPress={() => setConfirmLogout(true)} testID="account-logout" />
      </View>
      <View style={styles.group}>
        <ListRow
          icon="trash-outline"
          title="Supprimer mon compte"
          subtitle="Définitif : profil, photos, matchs et messages"
          danger
          onPress={() => setDeleting(true)}
          testID="account-delete"
        />
      </View>

      <Sheet visible={changing} title="Changer de mot de passe" onClose={() => setChanging(false)}>
        <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          error={nextError}
          hint="8 caractères minimum, avec une lettre et un chiffre."
        />
        {change.error ? (
          <Text variant="small" color="danger">
            {errorMessage(change.error)}
          </Text>
        ) : null}
        <Button
          label="Enregistrer"
          onPress={() => change.mutate()}
          disabled={!current || !next || !!nextError}
          loading={change.isPending}
        />
      </Sheet>

      <Dialog
        visible={confirmLogout}
        title="Se déconnecter ?"
        confirmLabel="Se déconnecter"
        onConfirm={() => {
          setConfirmLogout(false);
          void signOut();
        }}
        onCancel={() => setConfirmLogout(false)}
        testID="logout-dialog"
      />

      <Dialog
        visible={deleting}
        title="Supprimer définitivement votre compte ?"
        message="Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible."
        confirmLabel="Supprimer"
        destructive
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onCancel={() => setDeleting(false)}
        testID="delete-dialog"
      >
        <TextField
          label="Confirmez avec votre mot de passe"
          value={deletePassword}
          onChangeText={setDeletePassword}
          secureTextEntry
          testID="delete-password"
        />
        {remove.error ? (
          <Text variant="small" color="danger">
            {errorMessage(remove.error)}
          </Text>
        ) : null}
      </Dialog>
    </Screen>
  );
}

const styles = StyleSheet.create({
  email: {
    marginBottom: spacing.lg
  },
  group: {
    gap: spacing.xs,
    marginBottom: spacing.lg
  }
});
