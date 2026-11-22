import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { register } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { Checkbox } from "../../components/Checkbox";
import { useAuth } from "../../hooks/useAuth";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, spacing } from "../../ui";

export default function SignUpScreen({ navigation }: AuthScreenProps<"SignUp">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const pwError = touched ? passwordProblem(password) : null;

  const mutation = useMutation({
    mutationFn: () => register(normalizeEmail(email), password),
    onSuccess: signIn,
    onError: (err) => setError(errorMessage(err))
  });

  const submit = () => {
    setTouched(true);
    setError(null);
    if (!isValidEmail(email) || passwordProblem(password)) {
      return;
    }
    if (!adult) {
      setError("Vous devez avoir 18 ans ou plus pour utiliser Lueur.");
      return;
    }
    mutation.mutate();
  };

  return (
    <Screen
      footer={<Button label="Créer mon compte" loading={mutation.isPending} onPress={submit} testID="signup-submit" />}
      scroll
      testID="signup-screen"
    >
      <Header onBack={navigation.goBack} subtitle="Ça prend moins de deux minutes." title="Bienvenue" />
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          testID="signup-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="new-password"
          error={pwError}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          label="Mot de passe"
          onChangeText={setPassword}
          secureTextEntry
          testID="signup-password"
          textContentType="newPassword"
          value={password}
        />
        <Checkbox
          checked={adult}
          label="J'ai 18 ans ou plus et je m'engage à rester respectueux·se envers les autres membres."
          onChange={setAdult}
          testID="signup-adult"
        />
        {error ? (
          <Text accessibilityLiveRegion="polite" testID="signup-error" tone="danger" variant="caption">
            {error}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md
  }
});
