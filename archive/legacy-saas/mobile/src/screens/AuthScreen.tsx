import React, { useMemo, useState } from "react";
import { Pressable, StyleSheet } from "react-native";
import { useAuth, useLogin, useRegister } from "../hooks/useAuth";
import { ErrorView } from "../shared/feedback";
import { Button, Card, Input, Text, colors, spacing } from "../shared/ui";
import { Header, ScreenContainer } from "../shared/layout";

type Mode = "login" | "register";

export default function AuthScreen() {
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const { authError, clearAuthError } = useAuth();
  const loginMutation = useLogin();
  const registerMutation = useRegister();

  const title = useMemo(() => (mode === "login" ? "Welcome back" : "Create your workspace"), [mode]);
  const subtitle = useMemo(
    () =>
      mode === "login"
        ? "Sign in to continue into your shared SaaS workspace."
        : "Open a clean account experience with persistent team access.",
    [mode]
  );
  const switchLabel = mode === "login" ? "Need an account? Register" : "Already have an account? Login";
  const submitting = loginMutation.isPending || registerMutation.isPending;
  const surfaceError = formError ?? authError;

  async function onSubmit() {
    const normalizedEmail = email.trim().toLowerCase();
    if (!normalizedEmail || !password) {
      setFormError("email and password are required");
      return;
    }

    setFormError(null);
    clearAuthError();

    try {
      if (mode === "login") {
        await loginMutation.mutateAsync({ email: normalizedEmail, password });
      } else {
        await registerMutation.mutateAsync({ email: normalizedEmail, password });
      }
    } catch (submitError) {
      setFormError(submitError instanceof Error ? submitError.message : "authentication failed");
    }
  }

  return (
    <ScreenContainer centered contentMaxWidth={440}>
      <Card padding="xl" style={styles.card} variant="accent">
        <Header
          centered
          eyebrow="Go React SaaS"
          style={styles.header}
          subtitle={subtitle}
          title={title}
        />

        <Text style={styles.intro} tone="secondary">
          Minimal, secure access for a premium team workspace.
        </Text>

        <Input
          autoCapitalize="none"
          autoComplete="email"
          containerStyle={styles.field}
          helperText="Use the email tied to your workspace."
          keyboardType="email-address"
          label="Email"
          onChangeText={(value) => {
            setEmail(value);
            if (surfaceError) {
              setFormError(null);
              clearAuthError();
            }
          }}
          placeholder="Email"
          testID="auth-email-input"
          value={email}
        />

        <Input
          autoCapitalize="none"
          containerStyle={styles.field}
          error={formError?.toLowerCase().includes("password") ? formError : null}
          helperText={mode === "login" ? "Enter your secure password." : "Create a password for future sessions."}
          label="Password"
          onChangeText={(value) => {
            setPassword(value);
            if (surfaceError) {
              setFormError(null);
              clearAuthError();
            }
          }}
          placeholder="Password"
          secureTextEntry
          testID="auth-password-input"
          value={password}
        />

        {surfaceError ? (
          <ErrorView
            compact
            message={surfaceError}
            style={styles.error}
            title="Authentication issue"
          />
        ) : null}

        <Button
          fullWidth
          label={mode === "login" ? "Login" : "Register"}
          loading={submitting}
          onPress={onSubmit}
          testID="auth-submit-button"
        />

        <Pressable
          disabled={submitting}
          onPress={() => {
            setMode((prev) => (prev === "login" ? "register" : "login"));
            setFormError(null);
            clearAuthError();
          }}
          testID="auth-switch-mode-button"
        >
          <Text style={styles.switchText} tone="secondary" variant="label" weight="semibold">
            {switchLabel}
          </Text>
        </Pressable>
      </Card>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  card: {
    width: "100%"
  },
  header: {
    marginBottom: spacing.lg
  },
  intro: {
    marginBottom: spacing.xl,
    textAlign: "center"
  },
  field: {
    marginBottom: spacing.md
  },
  switchText: {
    marginTop: spacing.lg,
    textAlign: "center"
  },
  error: {
    marginBottom: spacing.sm
  }
});
