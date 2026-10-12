import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { forgotPassword, resetPassword } from "../api/auth";
import { errorMessage } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { validateEmail, validatePassword } from "../lib/validation";
import { APP_NAME, spacing } from "../theme";
import { Button, Input, Screen, Text, useToast } from "../ui";

type Mode = "login" | "register" | "forgot" | "reset";

export default function AuthScreen() {
  const { signIn, signUp } = useAuth();
  const toast = useToast();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<{
    email?: string | null;
    password?: string | null;
    confirm?: string | null;
    token?: string | null;
    form?: string | null;
  }>({});

  const switchMode = (next: Mode) => {
    setMode(next);
    setErrors({});
    setPassword("");
    setConfirm("");
  };

  const submit = async () => {
    const next: typeof errors = {};
    if (mode !== "reset") {
      next.email = validateEmail(email);
    }
    if (mode === "register" || mode === "reset") {
      next.password = validatePassword(password);
      if (!next.password && password !== confirm) {
        next.confirm = "Les mots de passe ne correspondent pas.";
      }
    } else if (mode === "login" && !password) {
      next.password = "Saisissez votre mot de passe.";
    }
    if (mode === "reset" && !token.trim()) {
      next.token = "Saisissez le code reçu par e-mail.";
    }
    setErrors(next);
    if (Object.values(next).some(Boolean)) {
      return;
    }

    setBusy(true);
    try {
      if (mode === "login") {
        await signIn(email, password);
      } else if (mode === "register") {
        await signUp(email, password);
      } else if (mode === "forgot") {
        await forgotPassword(email.trim());
        toast.show("Si un compte existe, un e-mail vient d'être envoyé.", "success");
        switchMode("reset");
      } else {
        await resetPassword(token.trim(), password);
        toast.show("Mot de passe modifié. Connectez-vous.", "success");
        switchMode("login");
      }
    } catch (error) {
      setErrors({ form: errorMessage(error) });
    } finally {
      setBusy(false);
    }
  };

  const titles: Record<Mode, string> = {
    login: "Bon retour",
    register: "Créer un compte",
    forgot: "Mot de passe oublié",
    reset: "Nouveau mot de passe"
  };
  const cta: Record<Mode, string> = {
    login: "Se connecter",
    register: "Créer mon compte",
    forgot: "Envoyer le lien",
    reset: "Changer le mot de passe"
  };

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Text variant="display" tone="primary">
          {APP_NAME}
        </Text>
        <Text tone="muted">Des rencontres sincères. Gratuit, pour toujours.</Text>
      </View>

      <Text variant="title">{titles[mode]}</Text>

      {mode !== "reset" ? (
        <Input
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
          error={errors.email}
          returnKeyType="next"
        />
      ) : (
        <Input
          label="Code reçu par e-mail"
          value={token}
          onChangeText={setToken}
          autoCapitalize="none"
          autoCorrect={false}
          error={errors.token}
        />
      )}

      {mode !== "forgot" ? (
        <Input
          label={mode === "reset" ? "Nouveau mot de passe" : "Mot de passe"}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={errors.password}
          hint={mode === "register" || mode === "reset" ? "8 caractères minimum" : undefined}
          onSubmitEditing={mode === "login" ? () => void submit() : undefined}
        />
      ) : null}

      {mode === "register" || mode === "reset" ? (
        <Input
          label="Confirmer le mot de passe"
          value={confirm}
          onChangeText={setConfirm}
          secureTextEntry
          autoCapitalize="none"
          error={errors.confirm}
        />
      ) : null}

      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert">
          {errors.form}
        </Text>
      ) : null}

      <Button label={cta[mode]} onPress={() => void submit()} loading={busy} />

      {mode === "register" ? (
        <Text variant="caption" tone="muted" style={styles.center}>
          En créant un compte, vous confirmez avoir 18 ans ou plus.
        </Text>
      ) : null}

      <View style={styles.links}>
        {mode === "login" ? (
          <>
            <Button label="Créer un compte" variant="ghost" onPress={() => switchMode("register")} />
            <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => switchMode("forgot")} />
          </>
        ) : (
          <>
            <Button label="J'ai déjà un compte" variant="ghost" onPress={() => switchMode("login")} />
            {mode === "forgot" ? <Button label="J'ai déjà un code" variant="ghost" onPress={() => switchMode("reset")} /> : null}
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { gap: spacing.xs, marginTop: spacing.xl },
  center: { textAlign: "center" },
  links: { alignItems: "center", gap: spacing.xs }
});
