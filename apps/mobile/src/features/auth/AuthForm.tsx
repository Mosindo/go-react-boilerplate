import React, { useState } from "react";
import { View } from "react-native";
import { Button, Input, Text } from "../../ui";
import { spacing } from "../../theme";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";

export type AuthFormProps = {
  mode: "login" | "register";
  onSubmit: (email: string, password: string) => Promise<void>;
  onForgotPassword?: () => void;
  onSwitchMode: () => void;
};

export function AuthForm({ mode, onSubmit, onForgotPassword, onSwitchMode }: AuthFormProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const isRegister = mode === "register";

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password:
        (isRegister ? validatePassword(password) : password ? null : "Enter your password.") ?? undefined
    };
    setErrors(next);
    setFormError(null);
    if (next.email || next.password) return;
    setBusy(true);
    try {
      await onSubmit(normalizeEmail(email), password);
    } catch (error) {
      setFormError(error instanceof Error ? error.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <View>
      <Input
        label="Email"
        testID="auth-email-input"
        value={email}
        onChangeText={setEmail}
        keyboardType="email-address"
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        error={errors.email}
      />
      <Input
        label="Password"
        testID="auth-password-input"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoCapitalize="none"
        autoComplete={isRegister ? "new-password" : "current-password"}
        error={errors.password}
      />
      {formError ? (
        <Text tone="danger" accessibilityRole="alert" style={{ marginBottom: spacing.md }}>
          {formError}
        </Text>
      ) : null}
      <Button
        testID="auth-submit-button"
        label={isRegister ? "Create account" : "Sign in"}
        onPress={() => void submit()}
        loading={busy}
      />
      {!isRegister && onForgotPassword ? (
        <Button label="Forgot your password?" variant="ghost" onPress={onForgotPassword} />
      ) : null}
      <Button
        testID="auth-switch-button"
        label={isRegister ? "I already have an account" : "Create an account"}
        variant="ghost"
        onPress={onSwitchMode}
      />
    </View>
  );
}
