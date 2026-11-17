import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import type { RootScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Screen, Text, TextField, showToast } from "../../ui";
import { confirmAction } from "../../utils/confirm";
import { passwordProblem } from "../../utils/format";

export function ChangePasswordScreen({ navigation }: RootScreenProps<"ChangePassword">) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      showToast("Mot de passe modifié. Tes autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    }
  });
  const problem = submitted ? passwordProblem(next) : null;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={styles.form}>
        <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry autoComplete="current-password" />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          autoComplete="new-password"
          error={problem}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
        />
        {change.isError ? <Text tone="danger">{errorMessage(change.error)}</Text> : null}
        <Button
          label="Changer le mot de passe"
          loading={change.isPending}
          onPress={() => {
            setSubmitted(true);
            if (current && !passwordProblem(next)) change.mutate();
          }}
        />
      </View>
    </Screen>
  );
}

export function DeleteAccountScreen() {
  const { signOut } = useAuth();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      await signOut({ remote: false });
      showToast("Ton compte et toutes tes données ont été supprimés.", "success");
    }
  });

  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="delete-account-screen">
      <View style={styles.form}>
        <Text variant="title">On est triste de te voir partir</Text>
        <Text tone="muted">
          La suppression est immédiate et définitive : profil, photos, likes, matchs, messages et notifications sont effacés. Les
          signalements te concernant sont conservés de façon anonyme pour la sécurité des membres.
        </Text>
        <Text tone="muted">Tu veux juste faire une pause ? Désactive plutôt « Apparaître dans Découvrir » dans Confidentialité.</Text>
        <TextField
          label="Confirme avec ton mot de passe"
          testID="delete-password"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="current-password"
        />
        {remove.isError ? <Text tone="danger">{errorMessage(remove.error)}</Text> : null}
        <Button
          label="Supprimer définitivement mon compte"
          variant="danger"
          testID="delete-confirm"
          disabled={!password}
          loading={remove.isPending}
          onPress={async () => {
            if (await confirmAction("Supprimer ton compte ?", "Cette action est irréversible.", "Supprimer")) remove.mutate();
          }}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.lg,
    paddingTop: spacing.md
  }
});
