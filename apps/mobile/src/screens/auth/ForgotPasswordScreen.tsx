import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { requestPasswordReset, resetPassword } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, showToast, spacing } from "../../ui";

export default function ForgotPasswordScreen({ navigation, route }: AuthScreenProps<"ForgotPassword">) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const request = useMutation({
    mutationFn: () => requestPasswordReset(normalizeEmail(email)),
    onSuccess: () => {
      setStep("code");
      setError(null);
    },
    onError: (err) => setError(errorMessage(err))
  });

  const reset = useMutation({
    mutationFn: () => resetPassword(normalizeEmail(email), code.trim(), password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("SignIn");
    },
    onError: (err) => setError(errorMessage(err))
  });

  const submit = () => {
    setError(null);
    if (step === "email") {
      if (!isValidEmail(email)) {
        setError("Adresse e-mail invalide.");
        return;
      }
      request.mutate();
      return;
    }
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    const problem = passwordProblem(password);
    if (problem) {
      setError(problem);
      return;
    }
    reset.mutate();
  };

  return (
    <Screen
      footer={
        <Button
          label={step === "email" ? "Recevoir un code" : "Changer mon mot de passe"}
          loading={request.isPending || reset.isPending}
          onPress={submit}
          testID="forgot-submit"
        />
      }
      scroll
    >
      <Header onBack={navigation.goBack} title="Mot de passe oublié" />
      <View style={styles.form}>
        {step === "email" ? (
          <>
            <Text tone="muted">Indiquez votre e-mail : si un compte existe, vous recevrez un code à 6 chiffres valable 15 minutes.</Text>
            <TextField
              autoCapitalize="none"
              inputMode="email"
              keyboardType="email-address"
              label="E-mail"
              onChangeText={setEmail}
              testID="forgot-email"
              value={email}
            />
          </>
        ) : (
          <>
            <Text tone="muted">Saisissez le code reçu à {normalizeEmail(email)} et choisissez un nouveau mot de passe.</Text>
            <TextField
              autoComplete="one-time-code"
              inputMode="numeric"
              keyboardType="number-pad"
              label="Code"
              maxLength={6}
              onChangeText={setCode}
              testID="forgot-code"
              textContentType="oneTimeCode"
              value={code}
            />
            <TextField
              autoComplete="new-password"
              label="Nouveau mot de passe"
              onChangeText={setPassword}
              secureTextEntry
              testID="forgot-password"
              value={password}
            />
            <Button compact label="Renvoyer un code" onPress={() => request.mutate()} variant="ghost" />
          </>
        )}
        {error ? (
          <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
            {error}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md
  }
});
