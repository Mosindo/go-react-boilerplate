import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import { isValidEmail, passwordProblem } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField, toast } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"email" | "code">("email");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const sendCode = async () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    const problem = passwordProblem(password);
    if (!/^\d{6}$/.test(code)) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await authApi.resetPassword(email.trim(), code, password);
      toast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Login");
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll edges={["top", "bottom"]} testID="forgot-screen">
      <Header onBack={() => navigation.goBack()} />
      <Text variant="title" style={styles.title} accessibilityRole="header">
        Récupérer votre compte
      </Text>
      {step === "email" ? (
        <View style={styles.form}>
          <Text color="textMuted">Nous vous enverrons un code à 6 chiffres, valable 15 minutes.</Text>
          <TextField
            label="E-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            autoComplete="email"
          />
          <InlineError message={error} />
          <Button label="Recevoir un code" onPress={sendCode} loading={loading} />
        </View>
      ) : (
        <View style={styles.form}>
          <Text color="textMuted">Si un compte existe pour {email.trim()}, un code vient d’être envoyé.</Text>
          <TextField
            label="Code"
            value={code}
            onChangeText={(v) => setCode(v.replace(/\D/g, "").slice(0, 6))}
            keyboardType="number-pad"
            autoComplete="one-time-code"
            textContentType="oneTimeCode"
          />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureToggle
            autoComplete="new-password"
            hint="8 caractères minimum, avec une lettre et un chiffre."
          />
          <InlineError message={error} />
          <Button label="Changer le mot de passe" onPress={reset} loading={loading} />
          <Button label="Renvoyer un code" variant="ghost" onPress={sendCode} disabled={loading} />
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { marginTop: spacing.md, marginBottom: spacing.lg },
  form: { gap: spacing.md }
});
