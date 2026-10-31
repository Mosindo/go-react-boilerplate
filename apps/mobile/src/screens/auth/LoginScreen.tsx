import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import { describeApiError, normalizeEmail, validateEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Notice } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export default function LoginScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => authApi.login(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const submit = () => {
    const emailError = validateEmail(email);
    if (emailError || !password) {
      setFieldError(emailError ?? "Saisissez votre mot de passe.");
      return;
    }
    setFieldError(null);
    mutation.mutate();
  };

  const serverError = mutation.error
    ? mutation.error instanceof ApiError
      ? describeApiError(mutation.error.status, mutation.error.message)
      : mutation.error.message
    : null;

  return (
    <AuthLayout subtitle="Heureux de vous revoir." title="Connexion">
      <Input
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        label="E-mail"
        onChangeText={setEmail}
        testID="login-email"
        value={email}
      />
      <Input
        autoComplete="password"
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        testID="login-password"
        value={password}
      />
      {fieldError || serverError ? <Notice title={fieldError ?? serverError ?? ""} tone="danger" /> : null}
      <Button label="Se connecter" loading={mutation.isPending} onPress={submit} testID="login-submit" />
      <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("ForgotPassword", { email })} variant="ghost" />
      <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} variant="outline" testID="goto-register" />
    </AuthLayout>
  );
}
