import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Screen, Text, TextField } from "../../ui";
import { isValidEmail } from "../../utils/format";
import type { AuthScreenProps } from "../../navigation/types";

export function LoginScreen({ navigation, route }: AuthScreenProps<"Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login(email.trim().toLowerCase(), password),
    onSuccess: signIn
  });

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const submit = () => {
    setTouched(true);
    if (isValidEmail(email) && password.length > 0) {
      login.mutate();
    }
  };

  return (
    <Screen scroll keyboard testID="login-screen">
      <Header onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <Text variant="title" accessibilityRole="header">
          Bon retour parmi nous
        </Text>
        {route.params?.notice ? <Text tone="success">{route.params.notice}</Text> : null}
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
          secure
          autoComplete="password"
          textContentType="password"
          onSubmitEditing={submit}
          testID="login-password"
        />
        <InlineError message={login.error ? errorMessage(login.error) : null} />
        <Button label="Se connecter" size="lg" loading={login.isPending} onPress={submit} testID="login-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword")} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, maxWidth: 480, width: "100%", alignSelf: "center", paddingTop: spacing.md }
});
