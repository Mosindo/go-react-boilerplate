import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { register } from "../../api";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import {
  normalizeEmail,
  validateEmail,
  validatePassword,
  validatePasswordConfirmation
} from "../../lib/validation";
import { useSession } from "../../session/SessionProvider";
import { Button, Text, TextField, useToast } from "../../ui";
import type { AuthStackParams } from "../navTypes";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParams, "Register">;

export default function RegisterScreen({ navigation }: Props) {
  const { signIn } = useSession();
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<Record<string, string | null>>({});

  const mutation = useMutation({
    mutationFn: () => register(normalizeEmail(email), password),
    onSuccess: signIn,
    onError: (error) => toast.error(errorMessage(error))
  });

  const submit = () => {
    const next = {
      email: validateEmail(email),
      password: validatePassword(password),
      confirm: validatePasswordConfirmation(password, confirm)
    };
    setErrors(next);
    if (!next.email && !next.password && !next.confirm) mutation.mutate();
  };

  return (
    <AuthLayout testID="register-screen" title={t.auth.registerTitle}>
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
        autoComplete="new-password"
        error={errors.password}
        hint={t.auth.passwordHint}
        label={t.auth.password}
        onChangeText={setPassword}
        secureTextEntry
        testID="auth-password-input"
        textContentType="newPassword"
        value={password}
      />
      <TextField
        autoCapitalize="none"
        error={errors.confirm}
        label={t.auth.confirmPassword}
        onChangeText={setConfirm}
        onSubmitEditing={submit}
        secureTextEntry
        testID="auth-confirm-input"
        value={confirm}
      />
      <Text variant="caption" center tone="muted">
        {t.auth.registerAdult}
      </Text>
      <Button
        label={t.auth.registerCta}
        loading={mutation.isPending}
        onPress={submit}
        testID="auth-submit-button"
      />
      <Button
        label={t.auth.haveAccount + " " + t.auth.loginCta}
        onPress={() => navigation.navigate("Login")}
        variant="ghost"
      />
    </AuthLayout>
  );
}
