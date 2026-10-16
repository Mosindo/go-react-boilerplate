import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { friendlyError } from "../api/errors";
import { useAuth, useLogin, useRegister } from "../hooks/useAuth";
import { normalizeEmail, validateEmail, validatePassword } from "../lib/validation";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Input, Notice, Text, colors, spacing } from "../shared/ui";

type Mode = "login" | "register" | "forgot" | "reset";

const TITLES: Record<Mode, { title: string; subtitle: string }> = {
  login: { title: "Bon retour", subtitle: "Connectez-vous pour retrouver vos matchs." },
  register: { title: "Créer un compte", subtitle: "Gratuit, sans abonnement ni limite." },
  forgot: { title: "Mot de passe oublié", subtitle: "Nous vous envoyons un code par e-mail." },
  reset: { title: "Nouveau mot de passe", subtitle: "Saisissez le code reçu par e-mail." }
};

export default function AuthScreen() {
  const { notice, clearNotice } = useAuth();
  const login = useLogin();
  const register = useRegister();

  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<{ email?: string | null; password?: string | null; code?: string | null }>({});
  const [message, setMessage] = useState<{ tone: "danger" | "success"; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const switchMode = (next: Mode) => {
    setMode(next);
    setErrors({});
    setMessage(null);
    clearNotice();
    setPassword("");
  };

  const submit = async () => {
    const emailError = validateEmail(email);
    const passwordError =
      mode === "login" ? (password ? null : "Saisissez votre mot de passe.") : mode === "forgot" ? null : validatePassword(password);
    const codeError = mode === "reset" && code.trim().length < 6 ? "Saisissez le code reçu par e-mail." : null;
    setErrors({ email: emailError, password: passwordError, code: codeError });
    setMessage(null);
    clearNotice();
    if (emailError || passwordError || codeError) {
      return;
    }
    const credentials = { email: normalizeEmail(email), password };

    try {
      if (mode === "login") {
        await login.mutateAsync(credentials);
      } else if (mode === "register") {
        await register.mutateAsync(credentials);
      } else if (mode === "forgot") {
        setBusy(true);
        await requestPasswordReset(credentials.email);
        setMode("reset");
        setMessage({ tone: "success", text: "Si un compte existe pour cette adresse, un code vient d'être envoyé." });
      } else {
        setBusy(true);
        await resetPassword(credentials.email, code.trim(), password);
        setMode("login");
        setPassword("");
        setCode("");
        setMessage({ tone: "success", text: "Mot de passe modifié. Vous pouvez vous connecter." });
      }
    } catch (error) {
      setMessage({ tone: "danger", text: friendlyError(error) });
    } finally {
      setBusy(false);
    }
  };

  const loading = busy || login.isPending || register.isPending;
  const copy = TITLES[mode];

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" testID="screen-auth">
          <View style={styles.brand}>
            <Text style={styles.logo} variant="title" weight="bold">
              Lien
            </Text>
            <Text tone="muted">Des rencontres sincères. Gratuites.</Text>
          </View>

          <View style={styles.form}>
            <Text variant="heading" weight="bold">
              {copy.title}
            </Text>
            <Text tone="muted">{copy.subtitle}</Text>

            {notice ? <Notice title={notice} tone="warning" /> : null}
            {message ? <Notice title={message.text} tone={message.tone} /> : null}

            <Input
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect={false}
              error={errors.email}
              keyboardType="email-address"
              label="Adresse e-mail"
              onChangeText={setEmail}
              placeholder="vous@exemple.fr"
              testID="input-email"
              textContentType="emailAddress"
              value={email}
            />
            {mode === "reset" ? (
              <Input
                autoCapitalize="characters"
                autoCorrect={false}
                error={errors.code}
                label="Code reçu par e-mail"
                maxLength={32}
                onChangeText={setCode}
                testID="input-code"
                value={code}
              />
            ) : null}
            {mode !== "forgot" ? (
              <Input
                autoCapitalize="none"
                autoComplete={mode === "login" ? "current-password" : "new-password"}
                error={errors.password}
                helperText={mode === "login" ? undefined : "8 caractères minimum."}
                label={mode === "reset" ? "Nouveau mot de passe" : "Mot de passe"}
                onChangeText={setPassword}
                secureTextEntry
                testID="input-password"
                textContentType={mode === "login" ? "password" : "newPassword"}
                value={password}
              />
            ) : null}

            <Button
              label={
                mode === "login"
                  ? "Se connecter"
                  : mode === "register"
                    ? "Créer mon compte"
                    : mode === "forgot"
                      ? "Envoyer le code"
                      : "Changer le mot de passe"
              }
              loading={loading}
              onPress={() => void submit()}
              size="lg"
              testID="auth-submit"
            />

            {mode === "register" ? (
              <Text tone="muted" variant="caption">
                En créant un compte, vous confirmez avoir au moins 18 ans. Votre âge est vérifié à la création de votre profil.
              </Text>
            ) : null}
          </View>

          <View style={styles.links}>
            {mode === "login" ? (
              <>
                <Button label="Créer un compte" onPress={() => switchMode("register")} testID="switch-register" variant="outline" />
                <Button label="Mot de passe oublié ?" onPress={() => switchMode("forgot")} testID="switch-forgot" variant="ghost" />
              </>
            ) : (
              <Button label="Retour à la connexion" onPress={() => switchMode("login")} testID="switch-login" variant="ghost" />
            )}
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { flexGrow: 1, padding: spacing.xl, gap: spacing.xxl, justifyContent: "center" },
  brand: { alignItems: "center", gap: spacing.xs },
  logo: { color: colors.primary, fontSize: 44, lineHeight: 52, letterSpacing: -1.5 },
  form: { gap: spacing.md },
  links: { gap: spacing.sm }
});
