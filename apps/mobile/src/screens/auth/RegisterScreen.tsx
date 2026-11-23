import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { register } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Input, Text } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

function passwordIssue(password: string): string | null {
  if (password.length === 0) {
    return null;
  }
  if (password.length < 8 || !/[A-Za-zÀ-ÿ]/.test(password) || !/\d/.test(password)) {
    return "8 caractères minimum, avec au moins une lettre et un chiffre.";
  }
  return null;
}

export default function RegisterScreen() {
  const { applySession } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const mutation = useMutation({
    mutationFn: () => register(email.trim(), password),
    onSuccess: applySession
  });
  const issue = passwordIssue(password);
  const mismatch = confirm.length > 0 && confirm !== password ? "Les mots de passe ne correspondent pas." : null;
  const canSubmit = /.+@.+\..+/.test(email.trim()) && password.length > 0 && !issue && confirm === password;

  return (
    <Screen testID="register-screen">
      <ScreenHeader subtitle="Ça prend une minute. C'est gratuit, pour toujours." title="Bienvenue" />
      <Input
        autoCapitalize="none"
        autoComplete="email"
        inputMode="email"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        testID="register-email"
        value={email}
      />
      <Input
        autoComplete="new-password"
        error={issue}
        helperText="8 caractères minimum, avec une lettre et un chiffre."
        label="Mot de passe"
        onChangeText={setPassword}
        secureTextEntry
        testID="register-password"
        value={password}
      />
      <Input
        autoComplete="new-password"
        error={mismatch ?? (mutation.isError ? errorMessage(mutation.error) : null)}
        label="Confirme le mot de passe"
        onChangeText={setConfirm}
        secureTextEntry
        testID="register-confirm"
        value={confirm}
      />
      <Text tone="subtle" variant="caption">
        En créant un compte, tu confirmes avoir 18 ans ou plus et t&apos;engages à respecter les autres membres.
      </Text>
      <Button
        disabled={!canSubmit}
        label="Créer mon compte"
        loading={mutation.isPending}
        onPress={() => mutation.mutate()}
        testID="register-submit"
      />
    </Screen>
  );
}
