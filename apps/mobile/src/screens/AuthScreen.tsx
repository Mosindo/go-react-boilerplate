import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { login, register } from "../api/auth";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import { isValidEmail, passwordProblem } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Input, Notice, Text, spacing, useTheme } from "../shared/ui";

type Mode = "login" | "register";

export default function AuthScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Auth">) {
  const { colors } = useTheme();
  const { signIn } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{ email?: string; password?: string }>({});

  const submit = useMutation({
    mutationFn: async () => (mode === "login" ? login(email.trim(), password) : register(email.trim(), password)),
    onSuccess: (session) => signIn(session)
  });

  const onSubmit = () => {
    const next: typeof fieldErrors = {};
    if (!isValidEmail(email)) {
      next.email = "Adresse e-mail invalide.";
    }
    if (mode === "register") {
      const problem = passwordProblem(password);
      if (problem) {
        next.password = problem;
      }
    } else if (password.length === 0) {
      next.password = "Saisissez votre mot de passe.";
    }
    setFieldErrors(next);
    if (Object.keys(next).length === 0) {
      submit.mutate();
    }
  };

  const switchMode = () => {
    setMode((current) => (current === "login" ? "register" : "login"));
    setFieldErrors({});
    submit.reset();
  };

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled" testID="auth-screen">
          <View style={styles.brand}>
            <Text style={{ color: colors.primary }} variant="title">
              Lien
            </Text>
            <Text style={styles.center} tone="muted">
              Des rencontres sincères. Gratuites, sans abonnement ni limite.
            </Text>
          </View>

          <View style={styles.form}>
            <Text variant="heading">{mode === "login" ? "Bon retour" : "Créer votre compte"}</Text>
            <Input
              autoCapitalize="none"
              autoComplete="email"
              error={fieldErrors.email}
              keyboardType="email-address"
              label="E-mail"
              onChangeText={setEmail}
              testID="auth-email-input"
              textContentType="emailAddress"
              value={email}
            />
            <Input
              autoCapitalize="none"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              error={fieldErrors.password}
              helperText={mode === "register" ? "8 caractères minimum." : undefined}
              label="Mot de passe"
              onChangeText={setPassword}
              onSubmitEditing={onSubmit}
              secureTextEntry
              testID="auth-password-input"
              textContentType={mode === "login" ? "password" : "newPassword"}
              value={password}
            />
            {submit.isError ? <Notice message={describeAuthError(submit.error, mode)} tone="danger" /> : null}
            <Button
              label={mode === "login" ? "Se connecter" : "Créer mon compte"}
              loading={submit.isPending}
              onPress={onSubmit}
              testID="auth-submit-button"
            />
            {mode === "register" ? (
              <Text style={styles.center} tone="muted" variant="caption">
                En créant un compte, vous confirmez avoir au moins 18 ans. Votre e-mail reste privé et n'est jamais montré aux autres
                membres.
              </Text>
            ) : (
              <Pressable accessibilityRole="link" onPress={() => navigation.navigate("ForgotPassword")} testID="auth-forgot-link">
                <Text style={styles.center} tone="primary" weight="semibold">
                  Mot de passe oublié ?
                </Text>
              </Pressable>
            )}
          </View>

          <Pressable accessibilityRole="button" onPress={switchMode} testID="auth-mode-toggle">
            <Text style={styles.center} tone="muted">
              {mode === "login" ? "Pas encore de compte ? " : "Déjà inscrit ? "}
              <Text tone="primary" weight="semibold">
                {mode === "login" ? "Créer un compte" : "Se connecter"}
              </Text>
            </Text>
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

function describeAuthError(error: unknown, mode: Mode): string {
  const e = error as { status?: number; message?: string };
  if (mode === "login" && e.status === 401) {
    return "E-mail ou mot de passe incorrect.";
  }
  return describeError(error);
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: {
    flexGrow: 1,
    justifyContent: "center",
    gap: spacing.xxl,
    padding: spacing.xl,
    maxWidth: 480,
    width: "100%",
    alignSelf: "center"
  },
  brand: { alignItems: "center", gap: spacing.sm },
  form: { gap: spacing.lg },
  center: { textAlign: "center" }
});
