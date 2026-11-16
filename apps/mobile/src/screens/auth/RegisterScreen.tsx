import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { errorMessage } from "../../utils/errors";

export function passwordIssue(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum";
  }
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) {
    return "Au moins une lettre et un chiffre";
  }
  return null;
}

export default function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmAdult, setConfirmAdult] = useState(false);
  const register = useMutation({ mutationFn: () => authApi.register(email.trim(), password), onSuccess: signIn });
  const issue = password ? passwordIssue(password) : null;
  const canSubmit = email.includes("@") && !passwordIssue(password) && confirmAdult;

  return (
    <Screen scroll keyboard testID="register-screen">
      <Header title="Créer un compte" onBack={navigation.goBack} large />
      <View style={styles.form}>
        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="register-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec une lettre et un chiffre"
          error={issue}
          testID="register-password"
        />
        <Button
          label={confirmAdult ? "J'ai 18 ans ou plus ✓" : "Je confirme avoir 18 ans ou plus"}
          variant={confirmAdult ? "secondary" : "ghost"}
          icon={confirmAdult ? "checkbox" : "square-outline"}
          onPress={() => setConfirmAdult((v) => !v)}
          testID="register-adult"
        />
        {register.error ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="register-error">
            {errorMessage(register.error)}
          </Text>
        ) : null}
        <Button label="Continuer" onPress={() => register.mutate()} loading={register.isPending} disabled={!canSubmit} testID="register-submit" />
        <Text variant="caption" tone="subtle" align="center">
          Lueur est gratuit, sans publicité ni abonnement. Tes données ne sont jamais vendues.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
