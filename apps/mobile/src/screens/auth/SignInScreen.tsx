import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { login } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { isValidEmail, normalizeEmail } from "../../lib/validation";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, spacing } from "../../ui";

export default function SignInScreen({ navigation }: AuthScreenProps<"SignIn">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => login(normalizeEmail(email), password),
    onSuccess: signIn,
    onError: (err) => setError(errorMessage(err))
  });

  const submit = () => {
    setError(null);
    if (!isValidEmail(email) || !password) {
      setError("Renseignez votre e-mail et votre mot de passe.");
      return;
    }
    mutation.mutate();
  };

  return (
    <Screen
      footer={<Button label="Se connecter" loading={mutation.isPending} onPress={submit} testID="signin-submit" />}
      scroll
      testID="signin-screen"
    >
      <Header onBack={navigation.goBack} title="Bon retour" />
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          testID="signin-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="current-password"
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureTextEntry
          testID="signin-password"
          textContentType="password"
          value={password}
        />
        {error ? (
          <Text accessibilityLiveRegion="polite" testID="signin-error" tone="danger" variant="caption">
            {error}
          </Text>
        ) : null}
        <Pressable
          accessibilityRole="link"
          onPress={() => navigation.navigate("ForgotPassword", { email: normalizeEmail(email) })}
          style={styles.link}
        >
          <Text tone="accent" variant="label">
            Mot de passe oublié ?
          </Text>
        </Pressable>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md
  },
  link: {
    alignSelf: "flex-start",
    paddingVertical: spacing.xs
  }
});
