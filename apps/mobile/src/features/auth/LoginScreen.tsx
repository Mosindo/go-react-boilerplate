import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { getErrorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { normalizeEmail, validateEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Text } from "../../ui";
import { AuthLayout } from "./AuthLayout";

export function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Login">) {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  async function submit() {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: password ? undefined : "Enter your password."
    };
    if (next.email || next.password) {
      setErrors(next);
      return;
    }
    setErrors({});
    setBusy(true);
    try {
      await login(normalizeEmail(email), password);
    } catch (error) {
      setErrors({ form: getErrorMessage(error) });
      setBusy(false);
    }
  }

  return (
    <AuthLayout subtitle="Sign in to pick up where you left off." title="Welcome back">
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
        autoComplete="current-password"
        error={errors.password}
        label="Password"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        passwordToggle
        returnKeyType="go"
        secureTextEntry
        textContentType="password"
        value={password}
      />
      {errors.form ? (
        <Text accessibilityRole="alert" tone="danger">
          {errors.form}
        </Text>
      ) : null}
      <Button label="Sign in" loading={busy} onPress={submit} />
      <Button label="Forgot password?" onPress={() => navigation.navigate("ForgotPassword")} variant="ghost" />
      <View style={{ height: 8 }} />
      <Text center tone="muted">
        New to Lumen?
      </Text>
      <Button label="Create an account" onPress={() => navigation.navigate("Register")} variant="secondary" />
    </AuthLayout>
  );
}
