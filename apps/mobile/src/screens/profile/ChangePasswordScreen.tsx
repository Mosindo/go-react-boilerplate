import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { passwordIssue } from "../auth/RegisterScreen";

export default function ChangePasswordScreen({ navigation }: AppScreenProps<"ChangePassword">) {
  const { forgetSession } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: async () => {
      showToast("Mot de passe modifié. Reconnecte-toi sur tes appareils.", "success");
      // Every session (including this one) is revoked server-side.
      await forgetSession();
    }
  });
  return (
    <Screen scroll keyboard>
      <Header title="Changer de mot de passe" onBack={navigation.goBack} />
      <View style={styles.form}>
        <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry autoComplete="password" />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          autoComplete="new-password"
          error={next ? passwordIssue(next) : null}
        />
        {change.error ? <Text tone="danger">{errorMessage(change.error)}</Text> : null}
        <Button label="Modifier" onPress={() => change.mutate()} loading={change.isPending} disabled={!current || !!passwordIssue(next)} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
