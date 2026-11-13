import { useMutation } from "@tanstack/react-query";
import React, { useState } from "react";
import { View } from "react-native";
import { authApi } from "../../api/auth";
import type { RootScreenProps } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { describeError } from "../../utils/errors";

export default function LoginScreen({ navigation }: RootScreenProps<"Login">) {
  const { spacing } = useTheme();
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useMutation({
    mutationFn: () => authApi.login(email.trim(), password),
    onSuccess: signIn
  });
  const canSubmit = email.trim().length > 3 && password.length > 0;

  return (
    <Screen scroll testID="login-screen">
      <Header back />
      <View style={{ gap: spacing.lg, marginTop: spacing.md }}>
        <Text variant="title" accessibilityRole="header">
          Bon retour parmi nous
        </Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="login-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="password"
          textContentType="password"
          onSubmitEditing={() => canSubmit && login.mutate()}
          testID="login-password"
        />
        {login.error ? (
          <Text tone="danger" testID="login-error">
            {describeError(login.error)}
          </Text>
        ) : null}
        <Button
          label="Se connecter"
          onPress={() => login.mutate()}
          loading={login.isPending}
          disabled={!canSubmit}
          testID="login-submit"
        />
        <Button
          label="Mot de passe oublié ?"
          variant="ghost"
          size="md"
          onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() })}
          testID="login-forgot"
        />
      </View>
    </Screen>
  );
}
