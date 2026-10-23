import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { forgotPassword } from "../../api";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { normalizeEmail, validateEmail } from "../../lib/validation";
import { Button, TextField, useToast } from "../../ui";
import type { AuthStackParams } from "../navTypes";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParams, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation }: Props) {
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => forgotPassword(normalizeEmail(email)),
    onSuccess: () => {
      toast.success(t.auth.forgotSent);
      navigation.navigate("ResetPassword", { email: normalizeEmail(email) });
    },
    onError: (err) => toast.error(errorMessage(err))
  });

  const submit = () => {
    const problem = validateEmail(email);
    setError(problem);
    if (!problem) mutation.mutate();
  };

  return (
    <AuthLayout subtitle={t.auth.forgotText} testID="forgot-screen" title={t.auth.forgotTitle}>
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        error={error}
        keyboardType="email-address"
        label={t.auth.email}
        onChangeText={setEmail}
        onSubmitEditing={submit}
        testID="auth-email-input"
        value={email}
      />
      <Button
        label={t.auth.forgotCta}
        loading={mutation.isPending}
        onPress={submit}
        testID="auth-submit-button"
      />
      <Button
        label={t.auth.backToLogin}
        onPress={() => navigation.navigate("Login")}
        variant="ghost"
      />
    </AuthLayout>
  );
}
