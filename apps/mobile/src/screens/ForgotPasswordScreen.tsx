import { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../api";
import { useToast } from "../components/feedback";
import { Button, Screen, Text, TextField } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { validateEmail, validatePassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { spacing } from "../theme";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation }: Props) {
  const toast = useToast();
  const [step, setStep] = useState<"request" | "reset">("request");
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const request = async () => {
    const problem = validateEmail(email);
    setError(problem);
    if (problem) {
      return;
    }
    setBusy(true);
    try {
      await authApi.forgotPassword(email.trim().toLowerCase());
      toast.show("Si un compte existe, un e-mail vient d'être envoyé.", "success");
      setStep("reset");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const problem = validatePassword(password) ?? (token.trim() ? null : "Code requis");
    setError(problem);
    if (problem) {
      return;
    }
    setBusy(true);
    try {
      await authApi.resetPassword(token.trim(), password);
      toast.show("Mot de passe mis à jour. Connectez-vous.", "success");
      navigation.navigate("Auth", { mode: "login" });
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen edges={["bottom"]} scroll>
      {step === "request" ? (
        <>
          <Text muted style={{ marginBottom: spacing.lg }}>
            Saisissez l'adresse de votre compte. Nous vous enverrons un code de réinitialisation valable 1 heure.
          </Text>
          <TextField
            autoCapitalize="none"
            error={error}
            keyboardType="email-address"
            label="E-mail"
            onChangeText={setEmail}
            value={email}
          />
          <Button loading={busy} onPress={request} title="Envoyer le code" />
        </>
      ) : (
        <>
          <Text muted style={{ marginBottom: spacing.lg }}>
            Collez le code reçu par e-mail et choisissez un nouveau mot de passe.
          </Text>
          <TextField autoCapitalize="none" label="Code" onChangeText={setToken} value={token} />
          <TextField
            autoCapitalize="none"
            error={error}
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secure
            value={password}
          />
          <Button loading={busy} onPress={reset} title="Réinitialiser" />
        </>
      )}
    </Screen>
  );
}
