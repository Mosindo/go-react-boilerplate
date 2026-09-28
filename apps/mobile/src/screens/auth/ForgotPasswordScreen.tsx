import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { forgotPassword, resetPassword } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { isValidEmail, normalizeEmail, passwordError as validatePassword } from "../../domain/validation";
import type { AuthScreenProps } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { Button, Input, Notice } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

export default function ForgotPasswordScreen({ navigation, route }: AuthScreenProps<"ForgotPassword">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"request" | "reset">("request");
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [showErrors, setShowErrors] = useState(false);

  const request = useMutation({
    mutationFn: () => forgotPassword(normalizeEmail(email)),
    onSuccess: () => {
      setStep("reset");
      setShowErrors(false);
    }
  });
  const reset = useMutation({
    mutationFn: () => resetPassword(normalizeEmail(email), code.trim(), newPassword),
    onSuccess: () => {
      showToast("Password updated. Sign in with your new password.", "success");
      navigation.navigate("Login");
    }
  });

  const submitRequest = () => {
    setShowErrors(true);
    if (isValidEmail(email)) {
      request.mutate();
    }
  };
  const submitReset = () => {
    setShowErrors(true);
    if (code.trim().length > 0 && !validatePassword(newPassword)) {
      reset.mutate();
    }
  };

  if (step === "request") {
    return (
      <AuthLayout subtitle="Enter your email and we will send you a code." title="Reset your password">
        {request.isError ? <Notice kind="error" message={errorMessage(request.error)} /> : null}
        <Input
          autoCapitalize="none"
          autoCorrect={false}
          error={showErrors && !isValidEmail(email) ? "Enter a valid email address." : null}
          keyboardType="email-address"
          label="Email"
          onChangeText={setEmail}
          value={email}
        />
        <Button label="Send code" loading={request.isPending} onPress={submitRequest} />
      </AuthLayout>
    );
  }

  return (
    <AuthLayout subtitle={`If an account exists for ${email.trim()}, a code is on its way.`} title="Enter your code">
      {reset.isError ? (
        <Notice kind="error" message={errorMessage(reset.error, "That code is invalid or has expired.")} />
      ) : null}
      <Input
        autoCapitalize="none"
        autoCorrect={false}
        error={showErrors && code.trim().length === 0 ? "Enter the code from your email." : null}
        keyboardType="number-pad"
        label="Code"
        onChangeText={setCode}
        value={code}
      />
      <Input
        autoComplete="new-password"
        error={showErrors ? validatePassword(newPassword) : null}
        hint="At least 8 characters. Changing it signs you out everywhere."
        label="New password"
        onChangeText={setNewPassword}
        onSubmitEditing={submitReset}
        secureTextEntry
        textContentType="newPassword"
        value={newPassword}
      />
      <Button label="Update password" loading={reset.isPending} onPress={submitReset} />
      <Button label="Use a different email" onPress={() => setStep("request")} variant="ghost" />
    </AuthLayout>
  );
}
