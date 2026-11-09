import React, { useState } from "react";
import { StyleSheet, View, KeyboardAvoidingView, Platform } from "react-native";
import { authApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthContext";
import { APP_NAME } from "../config";
import { describeError, isValidEmail, passwordProblem } from "../lib/validation";
import { Button, Input, Screen, Text, useToast } from "../ui";
import { useTheme } from "../theme/theme";

type Mode = "login" | "register" | "forgot" | "reset";

export default function AuthScreen() {
  const { login, register } = useAuth();
  const toast = useToast();
  const { colors } = useTheme();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [touched, setTouched] = useState(false);

  const emailError = touched && !isValidEmail(email) ? "Adresse email invalide." : null;
  const passwordError =
    touched && mode !== "forgot" && (mode === "login" ? (password ? null : "Mot de passe requis.") : passwordProblem(password));

  const switchMode = (next: Mode) => {
    setMode(next);
    setTouched(false);
  };

  const submit = async () => {
    setTouched(true);
    if (!isValidEmail(email) && mode !== "reset") return;
    setBusy(true);
    try {
      if (mode === "login") {
        if (!password) return;
        await login(email, password);
      } else if (mode === "register") {
        if (passwordProblem(password)) return;
        await register(email, password);
      } else if (mode === "forgot") {
        await authApi.forgotPassword(email.trim());
        toast.show("Si un compte existe, un code vous a été envoyé par email.");
        switchMode("reset");
      } else {
        if (!code.trim() || passwordProblem(password)) return;
        await authApi.resetPassword(code.trim(), password);
        toast.show("Mot de passe mis à jour. Connectez-vous.");
        setPassword("");
        switchMode("login");
      }
    } catch (error) {
      toast.show(describeError(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const titles: Record<Mode, string> = {
    login: "Content de vous revoir",
    register: "Créez votre compte",
    forgot: "Mot de passe oublié",
    reset: "Nouveau mot de passe"
  };
  const cta: Record<Mode, string> = {
    login: "Se connecter",
    register: "Créer mon compte",
    forgot: "Envoyer le code",
    reset: "Réinitialiser"
  };

  return (
    <Screen scroll contentStyle={styles.content}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.form}>
        <View style={styles.brand}>
          <Text variant="title" color={colors.primary}>
            {APP_NAME}
          </Text>
          <Text muted>Des rencontres sincères. Gratuites, pour toujours.</Text>
        </View>
        <Text variant="heading">{titles[mode]}</Text>

        {mode !== "reset" ? (
          <Input
            label="Email"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            textContentType="emailAddress"
            error={emailError}
            testID="auth-email"
          />
        ) : (
          <Input
            label="Code reçu par email"
            value={code}
            onChangeText={setCode}
            autoCapitalize="none"
            autoCorrect={false}
            testID="auth-code"
          />
        )}
        {mode !== "forgot" ? (
          <Input
            label={mode === "reset" ? "Nouveau mot de passe" : "Mot de passe"}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            textContentType={mode === "login" ? "password" : "newPassword"}
            error={passwordError || null}
            testID="auth-password"
          />
        ) : null}

        <Button title={cta[mode]} onPress={submit} loading={busy} testID="auth-submit" />

        {mode === "login" ? (
          <>
            <Button title="Mot de passe oublié ?" variant="ghost" onPress={() => switchMode("forgot")} />
            <Button
              title="Pas encore de compte ? S'inscrire"
              variant="secondary"
              onPress={() => switchMode("register")}
              testID="auth-switch-register"
            />
          </>
        ) : (
          <Button title="Retour à la connexion" variant="ghost" onPress={() => switchMode("login")} />
        )}
        {mode === "register" ? (
          <Text variant="caption" muted style={styles.legal}>
            En créant un compte vous confirmez avoir au moins 18 ans.
          </Text>
        ) : null}
      </KeyboardAvoidingView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { flexGrow: 1, justifyContent: "center" },
  form: { gap: 16 },
  brand: { gap: 4, marginBottom: 12 },
  legal: { textAlign: "center" }
});
