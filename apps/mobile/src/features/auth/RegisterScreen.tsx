import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { getErrorMessage, isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Text } from "../../ui";
import { AuthLayout } from "./AuthLayout";

export function RegisterScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Register">) {
  const { register } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; confirm?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  async function submit() {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      confirm: confirm !== password ? "Passwords don't match." : undefined
    };
    if (next.email || next.password || next.confirm) {
      setErrors(next);
      return;
    }
    setErrors({});
    setBusy(true);
    try {
      await register(normalizeEmail(email), password);
    } catch (error) {
      if (isApiError(error) && error.code === "conflict") {
        setErrors({ email: "An account with this email already exists. Try signing in." });
      } else if (isApiError(error) && error.fields.email) {
        setErrors({ email: error.fields.email });
      } else {
        setErrors({ form: getErrorMessage(error) });
      }
      setBusy(false);
    }
  }

  return (
    <AuthLayout subtitle="Free, with no limits. You must be 18 or older." title="Create your account">
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        error={errors.email}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        textContentType="emailAddress"
        value={email}
      />
      <Input
        autoCapitalize="none"
        autoComplete="new-password"
        error={errors.password}
        hint="At least 8 characters."
        label="Password"
        onChangeText={setPassword}
        passwordToggle
        secureTextEntry
        textContentType="newPassword"
        value={password}
      />
      <Input
        autoCapitalize="none"
        autoComplete="new-password"
        error={errors.confirm}
        label="Confirm password"
        onChangeText={setConfirm}
        onSubmitEditing={submit}
        secureTextEntry
        textContentType="newPassword"
        value={confirm}
      />
      {errors.form ? (
        <Text accessibilityRole="alert" tone="danger">
          {errors.form}
        </Text>
      ) : null}
      <Button label="Create account" loading={busy} onPress={submit} />
      <Button label="I already have an account" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </AuthLayout>
  );
}
