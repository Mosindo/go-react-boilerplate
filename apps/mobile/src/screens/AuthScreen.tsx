import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { login, register } from "../api/auth";
import { ApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { SafeAreaLayout } from "../shared/layout";
import { Button, FormField, Input, Notice, Text, spacing } from "../shared/ui";
import { isValidEmail, passwordProblem } from "../utils/validation";
import ForgotPasswordScreen from "./ForgotPasswordScreen";

type Mode = "login" | "register" | "forgot";

export default function AuthScreen() {
  const { signIn, sessionNotice, clearSessionNotice } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => (mode === "register" ? register(email.trim(), password) : login(email.trim(), password)),
    onSuccess: (session) => signIn(session),
    onError: (error) => setServerError(error instanceof ApiError ? error.message : "Une erreur est survenue.")
  });

  if (mode === "forgot") {
    return <ForgotPasswordScreen initialEmail={email} onBack={() => setMode("login")} />;
  }

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const passwordError =
    touched && mode === "register" ? passwordProblem(password) : touched && password.length === 0 ? "Mot de passe requis." : null;

  const submit = () => {
    setTouched(true);
    setServerError(null);
    clearSessionNotice();
    if (!isValidEmail(email) || (mode === "register" ? passwordProblem(password) : password.length === 0)) {
      return;
    }
    mutation.mutate();
  };

  const switchMode = (next: Mode) => {
    setMode(next);
    setServerError(null);
    setTouched(false);
  };

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <View style={styles.brand}>
            <Text tone="primary" variant="eyebrow" weight="bold">
              Lumen
            </Text>
            <Text accessibilityRole="header" variant="title">
              {mode === "login" ? "Content de vous revoir" : "Rejoignez Lumen"}
            </Text>
            <Text tone="muted">Des rencontres sincères. Gratuites, sans abonnement, sans limite de likes.</Text>
          </View>

          {sessionNotice ? <Notice message={sessionNotice} tone="warning" /> : null}

          <View style={styles.form}>
            <FormField error={emailError} label="Adresse e-mail">
              <Input
                accessibilityLabel="Adresse e-mail"
                testID="auth-email-input"
                autoCapitalize="none"
                autoComplete="email"
                autoCorrect={false}
                invalid={!!emailError}
                keyboardType="email-address"
                onChangeText={setEmail}
                placeholder="vous@exemple.fr"
                textContentType="emailAddress"
                value={email}
              />
            </FormField>
            <FormField error={passwordError} hint={mode === "register" ? "8 caractères minimum." : undefined} label="Mot de passe">
              <Input
                accessibilityLabel="Mot de passe"
                testID="auth-password-input"
                autoCapitalize="none"
                autoComplete={mode === "register" ? "new-password" : "current-password"}
                invalid={!!passwordError}
                onChangeText={setPassword}
                onSubmitEditing={submit}
                secureTextEntry
                textContentType={mode === "register" ? "newPassword" : "password"}
                value={password}
              />
            </FormField>
            {serverError ? <Notice message={serverError} tone="danger" /> : null}
            <Button
              label={mode === "login" ? "Se connecter" : "Créer mon compte"}
              loading={mutation.isPending}
              onPress={submit}
              testID="auth-submit-button"
            />
            {mode === "login" ? (
              <Pressable accessibilityRole="button" onPress={() => switchMode("forgot")} style={styles.link}>
                <Text tone="muted" variant="label" weight="semibold">
                  Mot de passe oublié ?
                </Text>
              </Pressable>
            ) : null}
          </View>

          <View style={styles.footer}>
            <Text tone="muted" variant="label">
              {mode === "login" ? "Pas encore de compte ?" : "Déjà inscrit ?"}
            </Text>
            <Button
              fullWidth={false}
              label={mode === "login" ? "Créer un compte" : "Se connecter"}
              onPress={() => switchMode(mode === "login" ? "register" : "login")}
              size="sm"
              variant="outline"
            />
            {mode === "register" ? (
              <Text align="center" tone="subtle" variant="caption">
                En créant un compte, vous confirmez avoir 18 ans ou plus.
              </Text>
            ) : null}
          </View>
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
  brand: { gap: spacing.sm },
  form: { gap: spacing.md },
  link: { alignSelf: "center", padding: spacing.sm },
  footer: { alignItems: "center", gap: spacing.sm }
});
