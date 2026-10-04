import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { getErrorMessage } from "../../api/client";
import { normalizeEmail, validateEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Text } from "../../ui";
import { AuthLayout } from "./AuthLayout";

export function ForgotPasswordScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "ForgotPassword">) {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);

  async function submit() {
    const problem = validateEmail(email);
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await authApi.forgotPassword(normalizeEmail(email));
      setSent(true);
    } catch (e) {
      setError(getErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthLayout
      subtitle="We'll email you a reset code if an account exists for that address."
      title="Reset your password"
    >
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        editable={!sent}
        error={error}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        value={email}
      />
      {sent ? (
        <Text accessibilityLiveRegion="polite" tone="success">
          If an account exists for that email, a reset code is on its way. Check your inbox, then enter the code.
        </Text>
      ) : (
        <Button label="Send reset code" loading={busy} onPress={submit} />
      )}
      <Button
        label="I have a reset code"
        onPress={() => navigation.navigate("ResetPassword", { email })}
        variant={sent ? "primary" : "secondary"}
      />
      <Button label="Back to sign in" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </AuthLayout>
  );
}
