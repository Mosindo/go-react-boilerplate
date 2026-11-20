import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";
import { useAuth } from "../../providers/AuthProvider";
import { useTheme } from "../../theme";
import { Button, Chip, Header, Screen, Text, TextField } from "../../ui";

export function RegisterScreen() {
  const { signIn } = useAuth();
  const { spacing } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Saisis une adresse e-mail valide." : null;
  const passwordError = submitted ? passwordProblem(password) : null;
  const adultError = submitted && !adult;

  const submit = () => {
    setSubmitted(true);
    if (isValidEmail(email) && !passwordProblem(password) && adult) {
      register.mutate();
    }
  };

  return (
    <Screen keyboard scroll header={<Header back title="Créer un compte" />}>
      <Text variant="title">Bienvenue sur Lueur</Text>
      <Text tone="muted">Crée ton compte en quelques secondes. Aucune carte bancaire, jamais.</Text>
      <View style={{ gap: spacing.md }}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          placeholder="toi@exemple.fr"
          testID="register-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="new-password"
          error={passwordError}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          label="Mot de passe"
          onChangeText={setPassword}
          secureTextEntry
          testID="register-password"
          textContentType="newPassword"
          value={password}
        />
        <View style={{ gap: spacing.xs }}>
          <View style={{ flexDirection: "row" }}>
            <Chip label="J’ai 18 ans ou plus" onPress={() => setAdult((v) => !v)} selected={adult} testID="register-adult" />
          </View>
          {adultError ? (
            <Text tone="danger" variant="caption">
              Lueur est réservée aux personnes majeures.
            </Text>
          ) : null}
        </View>
        {register.isError ? (
          <Text accessibilityLiveRegion="polite" tone="danger">
            {errorMessage(register.error)}
          </Text>
        ) : null}
        <Button label="Créer mon compte" loading={register.isPending} onPress={submit} testID="register-submit" />
        <Text center tone="subtle" variant="caption">
          En continuant, tu acceptes de respecter les autres membres. Tout comportement abusif peut être signalé.
        </Text>
      </View>
    </Screen>
  );
}
