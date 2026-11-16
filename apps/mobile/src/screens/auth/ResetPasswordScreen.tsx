import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { passwordIssue } from "./RegisterScreen";

export default function ResetPasswordScreen({ navigation, route }: AuthScreenProps<"ResetPassword">) {
  const { email } = route.params;
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(email, code.trim(), password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Connecte-toi avec le nouveau.", "success");
      navigation.navigate("Login");
    }
  });
  const issue = password ? passwordIssue(password) : null;
  return (
    <Screen scroll keyboard>
      <Header title="Nouveau mot de passe" onBack={navigation.goBack} large />
      <View style={styles.form}>
        <Text tone="muted">Saisis le code envoyé à {email}.</Text>
        <TextField
          label="Code"
          value={code}
          onChangeText={(v) => setCode(v.replace(/\D/g, "").slice(0, 6))}
          keyboardType="number-pad"
          autoComplete="one-time-code"
          textContentType="oneTimeCode"
          testID="reset-code"
        />
        <TextField
          label="Nouveau mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          error={issue}
          testID="reset-password"
        />
        {reset.error ? <Text tone="danger">{errorMessage(reset.error)}</Text> : null}
        <Button
          label="Changer le mot de passe"
          onPress={() => reset.mutate()}
          loading={reset.isPending}
          disabled={code.length !== 6 || !!passwordIssue(password)}
          testID="reset-submit"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
