import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { deleteAccount } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { confirm } from "../../lib/confirm";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, colors, radii, showToast, spacing } from "../../ui";

const erased = ["Votre profil, vos photos et vos préférences", "Vos likes, matchs et conversations", "Vos notifications et sessions"];

export default function DeleteAccountScreen({ navigation }: MainScreenProps<"DeleteAccount">) {
  const { forgetSession } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const submit = async () => {
    setError(null);
    if (!password) {
      setError("Confirmez avec votre mot de passe.");
      return;
    }
    if (!(await confirm("Supprimer définitivement ?", "Cette action est irréversible.", "Supprimer", true))) {
      return;
    }
    setDeleting(true);
    try {
      await deleteAccount(password);
      await forgetSession();
      showToast("Votre compte a été supprimé. Prenez soin de vous.", "success");
    } catch (err) {
      setError(errorMessage(err));
      setDeleting(false);
    }
  };

  return (
    <Screen
      footer={<Button label="Supprimer mon compte" loading={deleting} onPress={submit} testID="delete-confirm" variant="danger" />}
      scroll
    >
      <Header onBack={navigation.goBack} title="Supprimer mon compte" />
      <View style={styles.box}>
        <Text variant="label">Seront effacés immédiatement et définitivement :</Text>
        {erased.map((e) => (
          <Text key={e} tone="muted">
            • {e}
          </Text>
        ))}
        <Text tone="muted" variant="caption">
          Si vous souhaitez seulement faire une pause, désactivez plutôt « Apparaître dans la découverte » dans vos préférences.
        </Text>
      </View>
      <TextField
        autoComplete="current-password"
        error={error}
        label="Mot de passe"
        onChangeText={setPassword}
        secureTextEntry
        testID="delete-password"
        value={password}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  box: {
    gap: spacing.xs,
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.dangerSoft,
    marginBottom: spacing.lg
  }
});
