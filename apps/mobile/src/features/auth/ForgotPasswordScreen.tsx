import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { isValidEmail, passwordIssue } from "../../utils/validation";

export function ForgotPasswordScreen({ route }: AuthScreenProps<"ForgotPassword">) {
  const { signIn } = useAuth();
  const [step, setStep] = useState<"request" | "reset">("request");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const request = async () => {
    setError(null);
    if (!isValidEmail(email)) {
      setError("Adresse email invalide.");
      return;
    }
    setLoading(true);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("reset");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    setError(null);
    const issue = passwordIssue(password);
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    if (issue) {
      setError(issue);
      return;
    }
    setLoading(true);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      showToast("Mot de passe modifié. Bienvenue !", "success");
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll edges={["top", "bottom", "left", "right"]}>
      <Header back />
      <View style={styles.body}>
        <Text variant="title">Récupérer mon compte</Text>
        {step === "request" ? (
          <>
            <Text tone="muted">Indiquez l’email de votre compte : nous vous enverrons un code à 6 chiffres, valable 15 minutes.</Text>
            <TextField autoCapitalize="none" inputMode="email" keyboardType="email-address" label="Email" onChangeText={setEmail} value={email} />
          </>
        ) : (
          <>
            <Text tone="muted">Si un compte existe pour {email.trim()}, un code vient d’être envoyé. Pensez à vérifier vos spams.</Text>
            <TextField inputMode="numeric" keyboardType="number-pad" label="Code reçu" maxLength={6} onChangeText={setCode} value={code} />
            <TextField
              autoComplete="new-password"
              hint="8 caractères minimum, avec une lettre et un chiffre."
              label="Nouveau mot de passe"
              onChangeText={setPassword}
              secureTextEntry
              value={password}
            />
          </>
        )}
        {error ? (
          <Text tone="danger" variant="small">
            {error}
          </Text>
        ) : null}
        <Button label={step === "request" ? "Recevoir un code" : "Changer mon mot de passe"} loading={loading} onPress={step === "request" ? request : reset} />
        {step === "reset" ? <Button label="Renvoyer un code" onPress={request} variant="ghost" /> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ body: { gap: 16, paddingTop: 8 } });
