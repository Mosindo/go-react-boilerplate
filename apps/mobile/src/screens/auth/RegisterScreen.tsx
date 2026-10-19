import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthProvider";
import { errorMessage } from "../../lib/errors";
import { Button, Input, Screen, Text } from "../../ui";
import { spacing } from "../../theme/tokens";
import type { AuthStackParams } from "../../navigation/types";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateRegistration(
  email: string,
  password: string,
  confirm: string,
  adult: boolean
): string | null {
  if (!EMAIL_PATTERN.test(email.trim())) return "Saisissez une adresse e-mail valide.";
  if (password.length < 8) return "Le mot de passe doit contenir au moins 8 caractères.";
  if (new TextEncoder().encode(password).length > 72)
    return "Le mot de passe est trop long (72 octets maximum).";
  if (password !== confirm) return "Les mots de passe ne correspondent pas.";
  if (!adult) return "Vous devez confirmer avoir 18 ans ou plus.";
  return null;
}

export default function RegisterScreen(_props: NativeStackScreenProps<AuthStackParams, "Register">) {
  const { signUp } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [adult, setAdult] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    const problem = validateRegistration(email, password, confirm, adult);
    setError(problem);
    if (problem) return;
    setLoading(true);
    try {
      await signUp(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="register-screen">
      <Text variant="title">Créer votre compte</Text>
      <View style={{ gap: spacing.lg }}>
        <Input
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="auth-email-input"
        />
        <Input
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum."
          testID="auth-password-input"
        />
        <Input
          label="Confirmer le mot de passe"
          value={confirm}
          onChangeText={setConfirm}
          secureTextEntry
          textContentType="newPassword"
          testID="auth-confirm-input"
        />
        <Button
          label={adult ? "☑ Je confirme avoir 18 ans ou plus" : "☐ Je confirme avoir 18 ans ou plus"}
          variant="secondary"
          onPress={() => setAdult((v) => !v)}
          testID="auth-adult-toggle"
        />
        {error ? (
          <Text tone="danger" accessibilityRole="alert" testID="auth-error">
            {error}
          </Text>
        ) : null}
        <Button label="Continuer" onPress={submit} loading={loading} testID="auth-submit-button" />
      </View>
    </Screen>
  );
}
