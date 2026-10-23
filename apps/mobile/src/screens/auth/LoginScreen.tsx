import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { login } from "../../api";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { normalizeEmail, validateEmail, validateRequired } from "../../lib/validation";
import { useSession } from "../../session/SessionProvider";
import { Button, Text, TextField, useToast } from "../../ui";
import type { AuthStackParams } from "../navTypes";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParams, "Login">;

export default function LoginScreen({ navigation }: Props) {
  const { signIn } = useSession();
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string | null; password?: string | null }>({});

  const mutation = useMutation({
    mutationFn: () => login(normalizeEmail(email), password),
    onSuccess: signIn,
    onError: (error) => toast.error(errorMessage(error))
  });

  const submit = () => {
    const next = { email: validateEmail(email), password: validateRequired(password) };
    setErrors(next);
    if (!next.email && !next.password) mutation.mutate();
  };

  return (
    <AuthLayout subtitle={t.tagline} testID="login-screen" title={t.auth.loginTitle}>
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        error={errors.email}
        keyboardType="email-address"
        label={t.auth.email}
        onChangeText={setEmail}
        testID="auth-email-input"
        textContentType="emailAddress"
        value={email}
      />
      <TextField
        autoCapitalize="none"
        autoComplete="current-password"
        error={errors.password}
        label={t.auth.password}
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        testID="auth-password-input"
        textContentType="password"
        value={password}
      />
      <Button
        label={t.auth.loginCta}
        loading={mutation.isPending}
        onPress={submit}
        testID="auth-submit-button"
      />
      <Button
        label={t.auth.forgot}
        onPress={() => navigation.navigate("ForgotPassword")}
        testID="auth-forgot-link"
        variant="ghost"
      />
      <Text center tone="muted">
        {t.auth.noAccount}
      </Text>
      <Button
        label={t.auth.registerCta}
        onPress={() => navigation.navigate("Register")}
        testID="auth-register-link"
        variant="secondary"
      />
    </AuthLayout>
  );
}
