import React, { useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { errorMessage } from "../api/client";
import { forgotPassword, resetPassword } from "../api/auth";
import { isValidEmail, normalizeEmail, passwordIssue } from "../lib/validation";
import { ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";

/** Two steps: ask for a code by email, then choose a new password with that code. */
export default function ForgotPasswordScreen() {
  const navigation = useNavigation();
  const [step, setStep] = useState<"request" | "reset" | "done">("request");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const request = async () => {
    if (!isValidEmail(email)) {
      setError("Entrez une adresse email valide.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await forgotPassword(normalizeEmail(email));
      setStep("reset");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const issue = passwordIssue(password);
    if (!code.trim() || issue) {
      setError(issue ?? "Collez le code reçu par email.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await resetPassword(code.trim(), password);
      setStep("done");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer contentMaxWidth={440} testID="forgot-screen">
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text accessibilityRole="header" variant="title" weight="bold">
          Récupérer mon compte
        </Text>
        {step === "request" ? (
          <View style={styles.form}>
            <Text tone="muted">Nous vous envoyons un code à usage unique valable une heure.</Text>
            <Input
              autoCapitalize="none"
              keyboardType="email-address"
              label="Email"
              onChangeText={setEmail}
              testID="forgot-email"
              value={email}
            />
            {error ? <Notice description={error} title="Un instant" tone="danger" /> : null}
            <Button label="Envoyer le code" loading={busy} onPress={request} size="lg" testID="forgot-send" />
          </View>
        ) : null}
        {step === "reset" ? (
          <View style={styles.form}>
            <Notice
              description="Si un compte existe pour cette adresse, un code vient d'être envoyé."
              title="Vérifiez votre boîte mail"
              tone="success"
            />
            <Input autoCapitalize="none" label="Code reçu" onChangeText={setCode} testID="forgot-code" value={code} />
            <Input
              autoCapitalize="none"
              helperText="8 caractères minimum."
              label="Nouveau mot de passe"
              onChangeText={setPassword}
              secureTextEntry
              testID="forgot-password"
              value={password}
            />
            {error ? <Notice description={error} title="Un instant" tone="danger" /> : null}
            <Button label="Changer le mot de passe" loading={busy} onPress={reset} size="lg" testID="forgot-reset" />
          </View>
        ) : null}
        {step === "done" ? (
          <View style={styles.form}>
            <Notice
              description="Vous pouvez maintenant vous connecter avec votre nouveau mot de passe."
              title="Mot de passe modifié"
              tone="success"
            />
            <Button label="Retour à la connexion" onPress={() => navigation.goBack()} size="lg" />
          </View>
        ) : null}
        {step !== "done" ? <Button label="Retour" onPress={() => navigation.goBack()} variant="ghost" /> : null}
      </ScrollView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { flexGrow: 1, justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.xl },
  form: { gap: spacing.lg }
});
