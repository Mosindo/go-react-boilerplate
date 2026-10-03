import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { confirmPasswordReset, errorMessage, requestPasswordReset } from "../api/platform";
import { Screen } from "../components/Screen";
import { useAuth, useLogin, useRegister } from "../hooks/useAuth";
import { validateEmail, validatePassword } from "../lib/format";
import { showToast } from "../shared/feedback";
import { Button, Input, Notice, Text, colors, spacing } from "../shared/ui";

type Mode = "signin" | "register" | "forgot" | "reset";

export default function AuthScreen() {
  const { authNotice, clearAuthNotice } = useAuth();
  const login = useLogin();
  const register = useRegister();
  const [mode, setMode] = useState<Mode>("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const switchMode = (next: Mode) => {
    setMode(next);
    setError(null);
    clearAuthNotice();
  };

  const submit = async () => {
    setError(null);
    const emailError = validateEmail(email);
    if (emailError) {
      setError(emailError);
      return;
    }
    try {
      if (mode === "signin") {
        if (!password) {
          setError("Enter your password.");
          return;
        }
        await login.mutateAsync({ email, password });
      } else if (mode === "register") {
        const passwordError = validatePassword(password);
        if (passwordError) {
          setError(passwordError);
          return;
        }
        await register.mutateAsync({ email, password });
      } else if (mode === "forgot") {
        setBusy(true);
        await requestPasswordReset(email.trim());
        showToast("If that email has an account, a recovery code is on its way.", { tone: "success" });
        setPassword("");
        setMode("reset");
      } else {
        const passwordError = validatePassword(password);
        if (passwordError) {
          setError(passwordError);
          return;
        }
        if (!code.trim()) {
          setError("Paste the recovery code from your email.");
          return;
        }
        setBusy(true);
        await confirmPasswordReset(code.trim(), password);
        showToast("Password updated. You can sign in now.", { tone: "success" });
        setPassword("");
        setCode("");
        setMode("signin");
      }
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const loading = busy || login.isPending || register.isPending;
  const titles: Record<Mode, { title: string; subtitle: string; cta: string }> = {
    signin: { title: "Welcome back", subtitle: "Sign in to see who is waiting.", cta: "Sign in" },
    register: { title: "Create your account", subtitle: "Free forever. No subscriptions, no paywalls.", cta: "Create account" },
    forgot: { title: "Reset your password", subtitle: "We will email you a one-time recovery code.", cta: "Send code" },
    reset: { title: "Choose a new password", subtitle: "Enter the code from your email and a new password.", cta: "Update password" }
  };
  const copy = titles[mode];

  return (
    <Screen edges={["top", "right", "bottom", "left"]} testID="auth-screen">
      <View style={styles.brand}>
        <Text style={styles.logo} tone="primary" variant="heading" weight="bold">
          amora
        </Text>
      </View>
      <View style={styles.copy}>
        <Text variant="title" weight="bold">
          {copy.title}
        </Text>
        <Text tone="muted">{copy.subtitle}</Text>
      </View>

      {authNotice ? <Notice title={authNotice} tone="warning" /> : null}
      {error ? <Notice description={error} title="Check your details" tone="danger" /> : null}

      <View style={styles.form}>
        <Input
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          label="Email"
          onChangeText={setEmail}
          testID="auth-email"
          textContentType="emailAddress"
          value={email}
        />
        {mode === "reset" ? (
          <Input autoCapitalize="none" autoCorrect={false} label="Recovery code" onChangeText={setCode} testID="auth-code" value={code} />
        ) : null}
        {mode !== "forgot" ? (
          <Input
            autoCapitalize="none"
            autoComplete={mode === "signin" ? "current-password" : "new-password"}
            helperText={mode === "signin" ? undefined : "8 to 72 characters"}
            label={mode === "reset" ? "New password" : "Password"}
            onChangeText={setPassword}
            secureTextEntry
            testID="auth-password"
            textContentType={mode === "signin" ? "password" : "newPassword"}
            value={password}
          />
        ) : null}
        <Button label={copy.cta} loading={loading} onPress={() => void submit()} size="lg" testID="auth-submit" />
      </View>

      <View style={styles.links}>
        {mode === "signin" ? (
          <>
            <Button label="New here? Create an account" onPress={() => switchMode("register")} testID="auth-to-register" variant="ghost" />
            <Button label="Forgot your password?" onPress={() => switchMode("forgot")} testID="auth-to-forgot" variant="ghost" />
          </>
        ) : (
          <Button label="Back to sign in" onPress={() => switchMode("signin")} testID="auth-to-signin" variant="ghost" />
        )}
      </View>
      {mode === "register" ? (
        <Text style={styles.legal} tone="muted" variant="caption">
          By creating an account you confirm that you are at least 18 years old.
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  brand: { paddingTop: spacing.xl },
  logo: { fontSize: 28, letterSpacing: -1, color: colors.primary },
  copy: { gap: spacing.sm },
  form: { gap: spacing.lg },
  links: { alignItems: "center" },
  legal: { textAlign: "center" }
});
