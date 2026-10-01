import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { ApiError } from "../api/client";
import { SafeAreaLayout } from "../shared/layout";
import { Button, FormField, Input, Notice, Text, spacing } from "../shared/ui";
import { isValidEmail, passwordProblem } from "../utils/validation";

type Props = { initialEmail?: string; onBack: () => void };

/** Two steps: ask for a code by e-mail, then choose a new password with it. */
export default function ForgotPasswordScreen({ initialEmail = "", onBack }: Props) {
  const [step, setStep] = useState<"request" | "reset" | "done">("request");
  const [email, setEmail] = useState(initialEmail);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const onError = (e: unknown) => setError(e instanceof ApiError ? e.message : "Une erreur est survenue.");

  const request = useMutation({
    mutationFn: () => requestPasswordReset(email.trim()),
    onSuccess: () => {
      setError(null);
      setStep("reset");
    },
    onError
  });
  const reset = useMutation({
    mutationFn: () => resetPassword(code.trim(), password),
    onSuccess: () => {
      setError(null);
      setStep("done");
    },
    onError
  });

  const submitRequest = () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setError(null);
    request.mutate();
  };

  const submitReset = () => {
    const problem = code.trim().length < 6 ? "Saisissez le code reçu par e-mail." : passwordProblem(password);
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    reset.mutate();
  };

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <Text accessibilityRole="header" variant="title">
            Récupérer mon compte
          </Text>

          {step === "request" ? (
            <View style={styles.form}>
              <Text tone="muted">Indiquez votre adresse e-mail : nous vous enverrons un code valable 15 minutes.</Text>
              <FormField label="Adresse e-mail">
                <Input
                  accessibilityLabel="Adresse e-mail"
                  autoCapitalize="none"
                  autoComplete="email"
                  keyboardType="email-address"
                  onChangeText={setEmail}
                  value={email}
                />
              </FormField>
              {error ? <Notice message={error} tone="danger" /> : null}
              <Button label="Envoyer le code" loading={request.isPending} onPress={submitRequest} />
              <Button label="J'ai déjà un code" onPress={() => setStep("reset")} variant="ghost" />
            </View>
          ) : null}

          {step === "reset" ? (
            <View style={styles.form}>
              <Notice message="Si un compte existe pour cette adresse, un code vient d'être envoyé." />
              <FormField label="Code reçu par e-mail">
                <Input
                  accessibilityLabel="Code reçu par e-mail"
                  autoCapitalize="characters"
                  autoCorrect={false}
                  maxLength={16}
                  onChangeText={setCode}
                  value={code}
                />
              </FormField>
              <FormField hint="8 caractères minimum." label="Nouveau mot de passe">
                <Input
                  accessibilityLabel="Nouveau mot de passe"
                  autoCapitalize="none"
                  onChangeText={setPassword}
                  secureTextEntry
                  textContentType="newPassword"
                  value={password}
                />
              </FormField>
              {error ? <Notice message={error} tone="danger" /> : null}
              <Button label="Changer le mot de passe" loading={reset.isPending} onPress={submitReset} />
            </View>
          ) : null}

          {step === "done" ? (
            <View style={styles.form}>
              <Notice message="Mot de passe mis à jour. Vous pouvez vous connecter." tone="success" />
            </View>
          ) : null}

          <Button label={step === "done" ? "Retour à la connexion" : "Retour"} onPress={onBack} variant="outline" />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: {
    flexGrow: 1,
    justifyContent: "center",
    padding: spacing.xl,
    gap: spacing.xl,
    maxWidth: 520,
    width: "100%",
    alignSelf: "center"
  },
  form: { gap: spacing.md }
});
