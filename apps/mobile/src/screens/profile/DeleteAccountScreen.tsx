import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import type { AppScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";

export default function DeleteAccountScreen({ navigation }: AppScreenProps<"DeleteAccount">) {
  const { forgetSession } = useAuth();
  const [password, setPassword] = useState("");
  const del = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      await forgetSession();
      showToast("Ton compte et toutes tes données ont été supprimés.", "success");
    }
  });
  return (
    <Screen scroll keyboard testID="delete-account-screen">
      <Header title="Supprimer mon compte" onBack={navigation.goBack} />
      <View style={styles.form}>
        <View style={styles.warning}>
          <Text variant="bodyStrong" tone="danger">
            Action définitive
          </Text>
          <Text tone="muted">Ton profil, tes photos, tes matchs, tes messages et tes notifications seront supprimés immédiatement et définitivement.</Text>
          <Text tone="muted">Pour faire simplement une pause, désactive plutôt la visibilité de ton profil.</Text>
        </View>
        <TextField
          label="Confirme avec ton mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="password"
          testID="delete-password"
        />
        {del.error ? <Text tone="danger">{errorMessage(del.error)}</Text> : null}
        <Button
          label="Supprimer définitivement"
          variant="danger"
          onPress={() => del.mutate()}
          loading={del.isPending}
          disabled={!password}
          testID="delete-confirm"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  warning: {
    gap: spacing.sm,
    padding: spacing.lg,
    borderRadius: radii.lg,
    backgroundColor: colors.dangerSoft,
    borderWidth: 1,
    borderColor: "rgba(240,106,106,0.35)"
  }
});
