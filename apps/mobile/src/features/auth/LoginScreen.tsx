import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import type { RootScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Screen, Text, TextField } from "../../ui";
import { isValidEmail } from "../../utils/format";

export function LoginScreen({ navigation }: RootScreenProps<"Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login(email.trim(), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const passwordError = submitted && !password ? "Saisis ton mot de passe." : null;

  function submit() {
    setSubmitted(true);
    if (!isValidEmail(email) || !password) return;
    login.mutate();
  }

  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="login-screen">
      <View style={styles.form}>
        <Text variant="title">Bon retour 👋</Text>
        <TextField
          label="E-mail"
          testID="login-email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
        />
        <TextField
          label="Mot de passe"
          testID="login-password"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="current-password"
          textContentType="password"
          onSubmitEditing={submit}
          error={passwordError}
        />
        {login.isError ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="login-error">
            {errorMessage(login.error)}
          </Text>
        ) : null}
        <Button label="Se connecter" testID="login-submit" loading={login.isPending} onPress={submit} />
        <Button
          label="Mot de passe oublié ?"
          variant="ghost"
          onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() || undefined })}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.lg,
    paddingTop: spacing.lg
  }
});
