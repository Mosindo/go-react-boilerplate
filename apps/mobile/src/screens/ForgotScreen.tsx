import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../api/endpoints";
import { errorMessage } from "../hooks/queries";
import { validateEmail, validateNewPassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { Button, Input, Screen, Text, spacing, useToast } from "../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Forgot">;

export default function ForgotScreen({ navigation }: Props) {
  const toast = useToast();
  const [step, setStep] = useState<"email" | "reset">("email");
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const requestCode = async () => {
    const invalid = validateEmail(email);
    if (invalid) return setError(invalid);
    setError(null);
    setBusy(true);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("reset");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const invalid = validateNewPassword(password) ?? (token.trim() ? null : "Collez le code reçu par e-mail.");
    if (invalid) return setError(invalid);
    setError(null);
    setBusy(true);
    try {
      await authApi.resetPassword(token.trim(), password);
      toast.show("Mot de passe mis à jour. Connectez-vous.");
      navigation.navigate("Auth");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <View style={styles.form}>
        {step === "email" ? (
          <>
            <Text variant="title">Mot de passe oublié</Text>
            <Text tone="muted">
              Indiquez votre adresse e-mail : si un compte existe, nous vous enverrons un code valable 1 heure.
            </Text>
            <Input
              autoCapitalize="none"
              error={error}
              keyboardType="email-address"
              label="Adresse e-mail"
              onChangeText={setEmail}
              testID="forgot-email"
              value={email}
            />
            <Button label="Envoyer le code" loading={busy} onPress={requestCode} testID="forgot-send" />
          </>
        ) : (
          <>
            <Text variant="title">Nouveau mot de passe</Text>
            <Text tone="muted">
              Si un compte existe pour {email.trim()}, un code vient d'être envoyé. Collez-le ci-dessous.
            </Text>
            <Input
              autoCapitalize="none"
              label="Code reçu par e-mail"
              onChangeText={setToken}
              testID="reset-token"
              value={token}
            />
            <Input
              autoCapitalize="none"
              error={error}
              hint="10 caractères minimum."
              label="Nouveau mot de passe"
              onChangeText={setPassword}
              secureTextEntry
              testID="reset-password"
              value={password}
            />
            <Button label="Changer mon mot de passe" loading={busy} onPress={reset} testID="reset-submit" />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg, marginTop: spacing.xl } });
