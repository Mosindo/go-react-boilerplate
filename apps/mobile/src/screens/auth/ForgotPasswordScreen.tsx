import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { forgotPassword, resetPassword } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import { Button, Input, Screen, Text, useToast } from "../../ui";
import { spacing } from "../../theme/tokens";
import type { AuthStackParams } from "../../navigation/types";

export default function ForgotPasswordScreen({
  navigation,
  route
}: NativeStackScreenProps<AuthStackParams, "ForgotPassword">) {
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const requestCode = async () => {
    if (!email.trim()) {
      setError("Saisissez votre adresse e-mail.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await forgotPassword(email.trim());
      setSent(true);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const submitReset = async () => {
    if (code.trim().length < 6 || password.length < 8) {
      setError("Saisissez le code reçu et un nouveau mot de passe d'au moins 8 caractères.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await resetPassword(email.trim(), code.trim(), password);
      toast.show("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Login");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Text variant="title">Récupérer mon compte</Text>
      <View style={{ gap: spacing.lg }}>
        <Input
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          editable={!sent}
          testID="auth-email-input"
        />
        {sent ? (
          <>
            <Text tone="muted">
              Si un compte existe pour cette adresse, un code de récupération vient d&apos;être envoyé. Il est
              valable 1 heure.
            </Text>
            <Input
              label="Code de récupération"
              value={code}
              onChangeText={(v) => setCode(v.toUpperCase())}
              autoCapitalize="characters"
              autoCorrect={false}
              maxLength={16}
              testID="reset-code-input"
            />
            <Input
              label="Nouveau mot de passe"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              textContentType="newPassword"
              testID="reset-password-input"
            />
          </>
        ) : null}
        {error ? (
          <Text tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        {sent ? (
          <>
            <Button
              label="Modifier mon mot de passe"
              onPress={submitReset}
              loading={loading}
              testID="reset-submit"
            />
            <Button label="Renvoyer un code" variant="ghost" onPress={requestCode} disabled={loading} />
          </>
        ) : (
          <Button label="Recevoir un code" onPress={requestCode} loading={loading} testID="forgot-submit" />
        )}
      </View>
    </Screen>
  );
}
