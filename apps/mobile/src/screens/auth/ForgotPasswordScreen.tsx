import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { forgotPassword, resetPassword } from "../../api/auth";
import { showToast } from "../../shared/feedback";
import { validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { ScreenContainer } from "../../shared/layout";
import { Button, FormField, Input, Notice, Text } from "../../shared/ui";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation }: Props) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Une erreur est survenue.");
    } finally {
      setBusy(false);
    }
  };

  const sendCode = () => {
    const problem = validateEmail(email);
    if (problem) return setError(problem);
    void run(async () => {
      await forgotPassword(email.trim());
      setStep("code");
    });
  };

  const reset = () => {
    const problem = validatePassword(password);
    if (!code.trim()) return setError("Saisissez le code reçu par e-mail.");
    if (problem) return setError(problem);
    void run(async () => {
      await resetPassword(code.trim(), password);
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Auth");
    });
  };

  return (
    <ScreenContainer scroll contentStyle={{ maxWidth: 440 }} testID="forgot-screen">
      <Text variant="heading" accessibilityRole="header">
        Récupérer mon compte
      </Text>
      {step === "email" ? (
        <>
          <Text tone="muted">Indiquez votre e-mail : nous vous enverrons un code valable 30 minutes.</Text>
          {error ? <Notice message={error} /> : null}
          <FormField label="E-mail">
            <Input
              testID="forgot-email-input"
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              autoComplete="email"
              placeholder="vous@exemple.fr"
            />
          </FormField>
          <Button testID="forgot-send" label="Envoyer le code" loading={busy} onPress={sendCode} />
        </>
      ) : (
        <>
          <Notice tone="info" message="Si un compte existe pour cette adresse, un code vient d'être envoyé." />
          {error ? <Notice message={error} /> : null}
          <FormField label="Code reçu par e-mail">
            <Input
              testID="forgot-code-input"
              value={code}
              onChangeText={setCode}
              autoCapitalize="characters"
              autoCorrect={false}
              placeholder="XXXX-XXXX-XXXX-XXXX"
            />
          </FormField>
          <FormField label="Nouveau mot de passe" hint="8 caractères minimum.">
            <Input
              testID="forgot-password-input"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoCapitalize="none"
              autoComplete="new-password"
            />
          </FormField>
          <Button testID="forgot-reset" label="Changer mon mot de passe" loading={busy} onPress={reset} />
        </>
      )}
      <Button label="Retour à la connexion" variant="ghost" onPress={() => navigation.navigate("Auth")} />
    </ScreenContainer>
  );
}
