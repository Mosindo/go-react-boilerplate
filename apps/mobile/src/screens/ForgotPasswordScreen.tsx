import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { forgotPassword, resetPassword } from "../api/auth";
import { describeError } from "../api/client";
import type { AuthStackParamList } from "../navigation/types";
import { ScreenContainer } from "../shared/layout";
import { showToast } from "../shared/feedback";
import { Button, FormField, Notice, Text } from "../shared/ui";
import { validateEmail, validatePassword } from "../utils/validation";

export default function ForgotPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "Forgot">) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const requestCode = async () => {
    setTouched(true);
    setError(null);
    if (validateEmail(email)) return;
    setBusy(true);
    try {
      await forgotPassword(email.trim());
      setStep("code");
      setTouched(false);
    } catch (e) {
      setError(describeError(e));
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    setTouched(true);
    setError(null);
    if (!code.trim() || validatePassword(password)) return;
    setBusy(true);
    try {
      await resetPassword(email.trim(), code.trim(), password);
      showToast("Mot de passe mis à jour. Connectez-vous.", "success");
      navigation.popToTop();
    } catch (e) {
      setError(describeError(e));
      setBusy(false);
    }
  };

  return (
    <ScreenContainer>
      {step === "email" ? (
        <>
          <Text variant="heading">Récupérer mon compte</Text>
          <Text tone="muted">Saisissez l'adresse e-mail de votre compte. Nous vous enverrons un code valable 15 minutes.</Text>
          {error ? <Notice message={error} tone="error" /> : null}
          <FormField
            autoCapitalize="none"
            error={touched ? validateEmail(email) : null}
            keyboardType="email-address"
            label="Adresse e-mail"
            onChangeText={setEmail}
            value={email}
          />
          <Button fullWidth loading={busy} onPress={requestCode} size="lg">
            Envoyer le code
          </Button>
        </>
      ) : (
        <>
          <Text variant="heading">Saisissez votre code</Text>
          <Notice message="Si un compte existe pour cette adresse, un code vient d'être envoyé." />
          {error ? <Notice message={error} tone="error" /> : null}
          <FormField
            autoCapitalize="characters"
            autoCorrect={false}
            error={touched && !code.trim() ? "Saisissez le code reçu." : null}
            label="Code à 8 caractères"
            maxLength={8}
            onChangeText={setCode}
            value={code}
          />
          <FormField
            autoCapitalize="none"
            error={touched ? validatePassword(password) : null}
            hint="8 caractères minimum, avec des lettres et des chiffres ou symboles."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            value={password}
          />
          <Button fullWidth loading={busy} onPress={submitReset} size="lg">
            Changer le mot de passe
          </Button>
          <Button onPress={() => setStep("email")} variant="ghost">
            Renvoyer un code
          </Button>
        </>
      )}
    </ScreenContainer>
  );
}
