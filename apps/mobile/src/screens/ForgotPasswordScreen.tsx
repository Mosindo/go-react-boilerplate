import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { forgotPassword, resetPassword } from "../api/auth";
import { describeError } from "../lib/errors";
import { isValidEmail, passwordProblem } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";

export default function ForgotPasswordScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "ForgotPassword">) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [problem, setProblem] = useState<string | null>(null);

  const request = useMutation({
    mutationFn: () => forgotPassword(email.trim()),
    onSuccess: () => setStep("code")
  });

  const reset = useMutation({
    mutationFn: () => resetPassword({ email: email.trim(), code: code.trim(), newPassword: password }),
    onSuccess: () => {
      showToast("Mot de passe modifié. Connectez-vous.", "success");
      navigation.popToTop();
    }
  });

  const sendCode = () => {
    if (!isValidEmail(email)) {
      setProblem("Adresse e-mail invalide.");
      return;
    }
    setProblem(null);
    request.mutate();
  };

  const submitReset = () => {
    const passwordIssue = passwordProblem(password);
    if (code.trim().length < 6) {
      setProblem("Saisissez le code reçu par e-mail.");
    } else if (passwordIssue) {
      setProblem(passwordIssue);
    } else {
      setProblem(null);
      reset.mutate();
    }
  };

  const error = problem ?? (request.isError ? describeError(request.error) : null) ?? (reset.isError ? describeError(reset.error) : null);

  return (
    <SafeAreaLayout edges={["left", "right", "bottom"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          {step === "email" ? (
            <>
              <Text tone="muted">Indiquez l'e-mail de votre compte : nous vous enverrons un code à usage unique valable une heure.</Text>
              <Input
                autoCapitalize="none"
                keyboardType="email-address"
                label="E-mail"
                onChangeText={setEmail}
                testID="forgot-email-input"
                value={email}
              />
              {error ? <Notice message={error} tone="danger" /> : null}
              <Button label="Envoyer le code" loading={request.isPending} onPress={sendCode} testID="forgot-send" />
            </>
          ) : (
            <>
              <Notice message="Si un compte existe pour cet e-mail, un code vient d'être envoyé." tone="success" />
              <Input
                autoCapitalize="characters"
                label="Code reçu"
                maxLength={8}
                onChangeText={setCode}
                testID="forgot-code-input"
                value={code}
              />
              <Input
                autoCapitalize="none"
                helperText="8 caractères minimum."
                label="Nouveau mot de passe"
                onChangeText={setPassword}
                secureTextEntry
                testID="forgot-password-input"
                value={password}
              />
              {error ? <Notice message={error} tone="danger" /> : null}
              <Button label="Changer le mot de passe" loading={reset.isPending} onPress={submitReset} testID="forgot-reset" />
              <Button label="Renvoyer un code" onPress={() => setStep("email")} variant="ghost" />
            </>
          )}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { gap: spacing.lg, padding: spacing.xl, maxWidth: 480, width: "100%", alignSelf: "center" }
});
