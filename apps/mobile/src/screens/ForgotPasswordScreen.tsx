import React, { useState } from "react";
import { View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { errorMessage } from "../i18n/errors";
import type { RootStackParamList } from "../navigation/types";
import { ScreenContainer, Header } from "../shared/layout";
import { Button, FormField, Notice, Text, spacing } from "../shared/ui";
import { checkPassword, isValidEmail, normalizeEmail } from "../lib/validation";

/** Two steps: ask for a code by email, then choose a new password with that code. */
export default function ForgotPasswordScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const route = useRoute<RouteProp<RootStackParamList, "ForgotPassword">>();
  const [step, setStep] = useState<"email" | "code" | "done">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendCode = async () => {
    setError(null);
    if (!isValidEmail(email)) {
      setError("Saisissez une adresse e-mail valide.");
      return;
    }
    setBusy(true);
    try {
      await requestPasswordReset(normalizeEmail(email));
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    setError(null);
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    if (checkPassword(password)) {
      setError("Le nouveau mot de passe doit contenir entre 8 et 72 caractères.");
      return;
    }
    setBusy(true);
    try {
      await resetPassword(normalizeEmail(email), code.trim(), password);
      setStep("done");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer keyboardAware scroll testID="forgot-screen">
      <Header onBack={() => navigation.goBack()} title="Mot de passe oublié" />

      {step === "email" ? (
        <View style={{ gap: spacing.lg }}>
          <Text tone="muted">Indiquez l&apos;adresse de votre compte. Nous vous enverrons un code à 6 chiffres, valable 15 minutes.</Text>
          <FormField
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            label="Adresse e-mail"
            onChangeText={setEmail}
            testID="forgot-email-input"
            value={email}
          />
          {error ? <Notice message={error} tone="danger" /> : null}
          <Button fullWidth label="Recevoir un code" loading={busy} onPress={() => void sendCode()} size="lg" testID="forgot-send-button" />
        </View>
      ) : null}

      {step === "code" ? (
        <View style={{ gap: spacing.lg }}>
          <Notice message={`Si un compte existe pour ${normalizeEmail(email)}, un code vient d'être envoyé.`} tone="info" />
          <FormField
            keyboardType="number-pad"
            label="Code reçu par e-mail"
            maxLength={6}
            onChangeText={(text) => setCode(text.replace(/\D/g, ""))}
            testID="forgot-code-input"
            textContentType="oneTimeCode"
            value={code}
          />
          <FormField
            autoCapitalize="none"
            hint="Au moins 8 caractères."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            testID="forgot-password-input"
            textContentType="newPassword"
            value={password}
          />
          {error ? <Notice message={error} tone="danger" /> : null}
          <Button
            fullWidth
            label="Changer le mot de passe"
            loading={busy}
            onPress={() => void submitReset()}
            size="lg"
            testID="forgot-reset-button"
          />
          <Button label="Renvoyer un code" onPress={() => void sendCode()} size="sm" variant="ghost" />
        </View>
      ) : null}

      {step === "done" ? (
        <View style={{ gap: spacing.lg }}>
          <Notice
            message="Votre mot de passe a été modifié. Vous avez été déconnecté de tous vos appareils."
            title="C'est fait"
            tone="success"
          />
          <Button fullWidth label="Retour à la connexion" onPress={() => navigation.goBack()} size="lg" />
        </View>
      ) : null}
    </ScreenContainer>
  );
}
