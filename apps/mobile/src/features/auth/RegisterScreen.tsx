import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import type { RootScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { Button, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/format";

export function RegisterScreen(_props: RootScreenProps<"Register">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(email.trim(), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const passwordError = submitted ? passwordProblem(password) : null;

  function submit() {
    setSubmitted(true);
    if (!isValidEmail(email) || passwordProblem(password) || !adult) return;
    register.mutate();
  }

  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="register-screen">
      <View style={styles.form}>
        <Text variant="title">Bienvenue sur Lueur</Text>
        <Text tone="muted">Ton e-mail reste privé : il n’est jamais affiché sur ton profil.</Text>
        <TextField
          label="E-mail"
          testID="register-email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
        />
        <TextField
          label="Mot de passe"
          testID="register-password"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          error={passwordError}
        />
        <Pressable
          testID="register-adult"
          accessibilityRole="checkbox"
          aria-checked={adult}
          onPress={() => setAdult((v) => !v)}
          style={styles.checkRow}
        >
          <Ionicons name={adult ? "checkbox" : "square-outline"} size={24} color={adult ? colors.accent : colors.textMuted} />
          <Text style={styles.checkText}>
            J’ai 18 ans ou plus et j’accepte de respecter les autres membres (pas de harcèlement, pas de faux profil).
          </Text>
        </Pressable>
        {submitted && !adult ? (
          <Text variant="caption" tone="danger">
            Tu dois confirmer avoir 18 ans ou plus.
          </Text>
        ) : null}
        {register.isError ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="register-error">
            {errorMessage(register.error)}
          </Text>
        ) : null}
        <Button label="Créer mon compte" testID="register-submit" loading={register.isPending} onPress={submit} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.lg,
    paddingTop: spacing.lg
  },
  checkRow: {
    flexDirection: "row",
    gap: spacing.md,
    alignItems: "flex-start"
  },
  checkText: {
    flex: 1
  }
});
