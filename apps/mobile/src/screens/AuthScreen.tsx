import React, { useRef, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, type TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { errorMessage, isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { normalizeEmail, validateEmail, validatePassword } from "../lib/validation";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Input, Notice, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Mode = "login" | "register" | "forgot" | "reset";

export const APP_NAME = "Lueur";

const COPY: Record<Mode, { title: string; subtitle: string; cta: string }> = {
  login: { title: "Content de vous revoir", subtitle: "Connectez-vous pour retrouver vos matchs.", cta: "Se connecter" },
  register: { title: "Créer votre compte", subtitle: "Gratuit, sans abonnement, sans limite de likes.", cta: "Créer mon compte" },
  forgot: { title: "Mot de passe oublié", subtitle: "Nous vous envoyons un code de récupération par e-mail.", cta: "Envoyer le code" },
  reset: { title: "Nouveau mot de passe", subtitle: "Saisissez le code reçu et choisissez un nouveau mot de passe.", cta: "Réinitialiser" }
};

export default function AuthScreen() {
  const { colors } = useTheme();
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; code?: string }>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  function switchMode(next: Mode) {
    setMode(next);
    setErrors({});
    setFormError(null);
    setInfo(null);
    setPassword("");
  }

  async function submit() {
    const nextErrors: typeof errors = {};
    const emailError = validateEmail(email);
    if (emailError) {
      nextErrors.email = emailError;
    }
    if (mode === "register" || mode === "reset") {
      const passwordError = validatePassword(password);
      if (passwordError) {
        nextErrors.password = passwordError;
      }
    } else if (mode === "login" && !password) {
      nextErrors.password = "Saisissez votre mot de passe.";
    }
    if (mode === "reset" && code.trim().length < 6) {
      nextErrors.code = "Saisissez le code reçu par e-mail.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length > 0) {
      return;
    }

    setBusy(true);
    try {
      const normalized = normalizeEmail(email);
      if (mode === "login") {
        await signIn(normalized, password);
      } else if (mode === "register") {
        await signUp(normalized, password);
      } else if (mode === "forgot") {
        await requestPasswordReset(normalized);
        setMode("reset");
        setInfo("Si un compte existe pour cette adresse, un code vient d'être envoyé.");
      } else {
        await resetPassword(normalized, code.trim(), password);
        setMode("login");
        setInfo("Mot de passe modifié. Vous pouvez vous connecter.");
        setPassword("");
        setCode("");
      }
    } catch (error) {
      if (isApiError(error, 401) && mode === "login") {
        setFormError("E-mail ou mot de passe incorrect.");
      } else if (isApiError(error, 429)) {
        setFormError("Trop de tentatives. Patientez quelques minutes avant de réessayer.");
      } else {
        setFormError(errorMessage(error));
      }
    } finally {
      setBusy(false);
    }
  }

  const copy = COPY[mode];
  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <View style={styles.card}>
            <View style={styles.brand}>
              <View style={[styles.mark, { backgroundColor: colors.primary }]}>
                <Ionicons color={colors.primaryForeground} name="heart" size={26} />
              </View>
              <Text variant="heading" weight="bold">
                {APP_NAME}
              </Text>
            </View>

            <View style={styles.copy}>
              <Text accessibilityRole="header" variant="title" weight="bold">
                {copy.title}
              </Text>
              <Text tone="muted">{copy.subtitle}</Text>
            </View>

            {info ? <Notice message={info} tone="success" /> : null}
            {formError ? <Notice message={formError} tone="danger" /> : null}

            <View style={styles.form}>
              <Input
                autoCapitalize="none"
                autoComplete="email"
                error={errors.email}
                keyboardType="email-address"
                label="Adresse e-mail"
                onChangeText={setEmail}
                onSubmitEditing={() => (mode === "forgot" ? void submit() : passwordRef.current?.focus())}
                returnKeyType="next"
                testID="auth-email"
                textContentType="emailAddress"
                value={email}
              />
              {mode === "reset" ? (
                <Input
                  autoCapitalize="characters"
                  autoCorrect={false}
                  error={errors.code}
                  label="Code de récupération"
                  onChangeText={setCode}
                  testID="auth-code"
                  value={code}
                />
              ) : null}
              {mode !== "forgot" ? (
                <Input
                  autoCapitalize="none"
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  error={errors.password}
                  hint={mode === "login" ? undefined : "8 caractères minimum, avec une lettre et un chiffre."}
                  label={mode === "reset" ? "Nouveau mot de passe" : "Mot de passe"}
                  onChangeText={setPassword}
                  onSubmitEditing={() => void submit()}
                  ref={passwordRef}
                  returnKeyType="go"
                  secureTextEntry
                  testID="auth-password"
                  textContentType={mode === "login" ? "password" : "newPassword"}
                  value={password}
                />
              ) : null}
            </View>

            <Button fullWidth label={copy.cta} loading={busy} onPress={() => void submit()} size="lg" testID="auth-submit" />

            <View style={styles.links}>
              {mode === "login" ? (
                <>
                  <Button label="Mot de passe oublié ?" onPress={() => switchMode("forgot")} size="sm" variant="ghost" />
                  <Button
                    label="Pas encore de compte ? S'inscrire"
                    onPress={() => switchMode("register")}
                    size="sm"
                    variant="ghost"
                    testID="auth-switch-register"
                  />
                </>
              ) : (
                <Button label="Retour à la connexion" onPress={() => switchMode("login")} size="sm" variant="ghost" />
              )}
            </View>

            {mode === "register" ? (
              <Text style={styles.legal} tone="subtle" variant="caption">
                En créant un compte, vous confirmez avoir au moins 18 ans.
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
  scroll: { flexGrow: 1, justifyContent: "center", padding: spacing.lg },
  card: { width: "100%", maxWidth: 440, alignSelf: "center", gap: spacing.xl },
  brand: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  mark: { width: 44, height: 44, borderRadius: radii.md, alignItems: "center", justifyContent: "center" },
  copy: { gap: spacing.xs },
  form: { gap: spacing.md },
  links: { alignItems: "center", gap: spacing.xs },
  legal: { textAlign: "center" }
});
