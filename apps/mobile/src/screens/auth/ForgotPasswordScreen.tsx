import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { AppText, Banner, Button, Screen, TextField } from "../../components/ui";
import { authApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { validateEmail, validatePassword } from "../../utils/validation";
import type { AuthStackParams } from "../../navigation/types";

export default function ForgotPasswordScreen({
  navigation,
}: NativeStackScreenProps<AuthStackParams, "ForgotPassword">) {
  const [step, setStep] = useState<"request" | "reset" | "done">("request");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <AppText variant="title">Récupérer mon compte</AppText>
      {error ? <Banner message={error} /> : null}
      {step === "request" ? (
        <>
          <AppText variant="muted">
            Indiquez votre e-mail : si un compte existe, vous recevrez un code de réinitialisation.
          </AppText>
          <TextField
            testID="email"
            label="Adresse e-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
          />
          <Button
            testID="send-code"
            title="Envoyer le code"
            loading={busy}
            onPress={() => {
              const problem = validateEmail(email);
              if (problem) return setError(problem);
              void run(async () => {
                await authApi.forgotPassword(email.trim());
                setStep("reset");
              });
            }}
          />
        </>
      ) : null}
      {step === "reset" ? (
        <>
          <Banner
            tone="info"
            message="Si un compte existe, un e-mail vient d'être envoyé. Saisissez le code reçu."
          />
          <TextField
            label="Code de réinitialisation"
            value={code}
            onChangeText={setCode}
            autoCapitalize="none"
            autoCorrect={false}
          />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoCapitalize="none"
            hint="8 caractères minimum."
          />
          <Button
            title="Changer le mot de passe"
            loading={busy}
            onPress={() => {
              const problem = validatePassword(password) ?? (code.trim() ? null : "Saisissez le code reçu.");
              if (problem) return setError(problem);
              void run(async () => {
                await authApi.resetPassword(code.trim(), password);
                setStep("done");
              });
            }}
          />
        </>
      ) : null}
      {step === "done" ? (
        <>
          <Banner tone="info" message="Mot de passe mis à jour. Vous pouvez vous connecter." />
          <Button title="Retour à la connexion" onPress={() => navigation.popToTop()} />
        </>
      ) : null}
    </Screen>
  );
}
