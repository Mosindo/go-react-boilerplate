import React, { useState } from "react";
import { errorMessage } from "../../api/client";
import { isValidEmail } from "../../domain/validation";
import { useLogin } from "../../hooks/useAuthActions";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Input, Notice } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

export default function LoginScreen({ navigation }: AuthScreenProps<"Login">) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showErrors, setShowErrors] = useState(false);
  const login = useLogin();

  const emailError = showErrors && !isValidEmail(email) ? "Enter a valid email address." : null;
  const passwordError = showErrors && password.length === 0 ? "Enter your password." : null;

  const submit = () => {
    setShowErrors(true);
    if (!isValidEmail(email) || password.length === 0) {
      return;
    }
    login.mutate({ email, password });
  };

  return (
    <AuthLayout subtitle="Sign in to pick up where you left off." title="Welcome back">
      {login.isError ? <Notice kind="error" message={errorMessage(login.error, "Could not sign in.")} /> : null}
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        error={emailError}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        textContentType="emailAddress"
        value={email}
      />
      <Input
        autoComplete="current-password"
        error={passwordError}
        label="Password"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        textContentType="password"
        value={password}
      />
      <Button label="Sign in" loading={login.isPending} onPress={submit} />
      <Button
        label="Forgot your password?"
        onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() })}
        variant="ghost"
      />
    </AuthLayout>
  );
}
