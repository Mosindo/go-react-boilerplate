import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useSession } from "../../hooks/useSession";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { normalizeEmail, validateEmail } from "../../utils/validation";

export function LoginScreen({ navigation }: AuthScreenProps<"Login">) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const emailError = submitted ? validateEmail(email) : null;
  const passwordError = submitted && !password ? "Saisissez votre mot de passe." : null;

  const submit = () => {
    setSubmitted(true);
    if (validateEmail(email) || !password) {
      return;
    }
    login.mutate();
  };

  return (
    <Screen scroll testID="login-screen">
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Bon retour parmi nous</Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
          testID="login-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="password"
          textContentType="password"
          error={passwordError}
          onSubmitEditing={submit}
          testID="login-password"
        />
        {login.error ? (
          <Text variant="small" color="danger" accessibilityLiveRegion="polite" testID="login-error">
            {errorMessage(login.error)}
          </Text>
        ) : null}
        <Button label="Se connecter" onPress={submit} loading={login.isPending} testID="login-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword")} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md,
    paddingTop: spacing.md
  }
});
