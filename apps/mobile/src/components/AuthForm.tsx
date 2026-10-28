import React, { useState } from "react";
import { View } from "react-native";
import { validateEmail, validatePassword } from "../lib/validation";
import { spacing } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";
import { TextField } from "./TextField";

type Props = {
  submitLabel: string;
  loading?: boolean;
  error?: string | null;
  onSubmit: (email: string, password: string) => void;
  /** Password length rules only apply when creating an account. */
  strictPassword?: boolean;
};

export function AuthForm({ submitLabel, loading, error, onSubmit, strictPassword }: Props) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string | null; password?: string | null }>({});

  const submit = (): void => {
    const next = {
      email: validateEmail(email),
      password: strictPassword ? validatePassword(password) : password ? null : "Enter your password."
    };
    setErrors(next);
    if (!next.email && !next.password) {
      onSubmit(email.trim(), password);
    }
  };

  return (
    <View style={{ gap: spacing.md }}>
      <TextField
        label="Email"
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
        testID="auth-email-input"
      />
      <TextField
        label="Password"
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secureTextEntry
        autoComplete={strictPassword ? "new-password" : "current-password"}
        textContentType={strictPassword ? "newPassword" : "password"}
        hint={strictPassword ? "At least 8 characters." : undefined}
        testID="auth-password-input"
        onSubmitEditing={submit}
      />
      {error ? (
        <Text tone="danger" accessibilityRole="alert" testID="auth-error">
          {error}
        </Text>
      ) : null}
      <Button title={submitLabel} onPress={submit} loading={loading} testID="auth-submit-button" />
    </View>
  );
}
