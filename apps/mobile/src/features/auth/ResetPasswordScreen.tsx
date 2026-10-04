import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { getErrorMessage } from "../../api/client";
import { validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Text, toast } from "../../ui";
import { AuthLayout } from "./AuthLayout";

export function ResetPasswordScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "ResetPassword">) {
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ token?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  async function submit() {
    const next = {
      token: token.trim() ? undefined : "Paste the code from your email.",
      password: validatePassword(password) ?? undefined
    };
    if (next.token || next.password) {
      setErrors(next);
      return;
    }
    setErrors({});
    setBusy(true);
    try {
      await authApi.resetPassword(token.trim(), password);
      toast.success("Password updated. Sign in with your new password.");
      navigation.navigate("Login");
    } catch (e) {
      setErrors({ form: getErrorMessage(e, "That code is invalid or has expired.") });
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthLayout subtitle="Enter the code from your email and choose a new password." title="Choose a new password">
      <Input
        autoCapitalize="none"
        autoCorrect={false}
        error={errors.token}
        label="Reset code"
        onChangeText={setToken}
        value={token}
      />
      <Input
        autoCapitalize="none"
        error={errors.password}
        hint="At least 8 characters. Signing in elsewhere will be ended."
        label="New password"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        passwordToggle
        secureTextEntry
        textContentType="newPassword"
        value={password}
      />
      {errors.form ? (
        <Text accessibilityRole="alert" tone="danger">
          {errors.form}
        </Text>
      ) : null}
      <Button label="Update password" loading={busy} onPress={submit} />
      <Button label="Back to sign in" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </AuthLayout>
  );
}
