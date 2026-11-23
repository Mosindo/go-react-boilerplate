import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { login } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Input } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export default function LoginScreen({ navigation }: Props) {
  const { applySession } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const mutation = useMutation({
    mutationFn: () => login(email.trim(), password),
    onSuccess: applySession
  });
  const canSubmit = email.trim().length > 3 && password.length > 0;

  return (
    <Screen testID="login-screen">
      <ScreenHeader subtitle="Content de te revoir." title="Connexion" />
      <Input
        autoCapitalize="none"
        autoComplete="email"
        inputMode="email"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        testID="login-email"
        value={email}
      />
      <Input
        autoComplete="current-password"
        error={mutation.isError ? errorMessage(mutation.error) : null}
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={() => canSubmit && mutation.mutate()}
        secureTextEntry
        testID="login-password"
        value={password}
      />
      <Button
        disabled={!canSubmit}
        label="Se connecter"
        loading={mutation.isPending}
        onPress={() => mutation.mutate()}
        testID="login-submit"
      />
      <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("ForgotPassword", { email })} size="sm" variant="ghost" />
    </Screen>
  );
}
