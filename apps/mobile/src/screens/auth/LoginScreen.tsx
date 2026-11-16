import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { errorMessage } from "../../utils/errors";

export default function LoginScreen({ navigation }: AuthScreenProps<"Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useMutation({ mutationFn: () => authApi.login(email.trim(), password), onSuccess: signIn });
  const canSubmit = email.trim().length > 3 && password.length > 0;

  return (
    <Screen scroll keyboard testID="login-screen">
      <Header title="Connexion" onBack={navigation.goBack} large />
      <View style={styles.form}>
        <TextField
          label="Email"
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
          secureTextEntry
          autoComplete="password"
          textContentType="password"
          onSubmitEditing={() => canSubmit && login.mutate()}
          testID="login-password"
        />
        {login.error ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="login-error">
            {errorMessage(login.error)}
          </Text>
        ) : null}
        <Button label="Se connecter" onPress={() => login.mutate()} loading={login.isPending} disabled={!canSubmit} testID="login-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword")} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
