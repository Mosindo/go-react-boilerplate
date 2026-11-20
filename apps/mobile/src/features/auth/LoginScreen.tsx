import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { isValidEmail, normalizeEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { useAuth } from "../../providers/AuthProvider";
import { useTheme } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const { spacing } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Saisis une adresse e-mail valide." : null;
  const passwordError = submitted && !password ? "Saisis ton mot de passe." : null;

  const submit = () => {
    setSubmitted(true);
    if (isValidEmail(email) && password) {
      login.mutate();
    }
  };

  return (
    <Screen keyboard scroll header={<Header back title="Connexion" />}>
      <Text variant="title">Bon retour parmi nous</Text>
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
          testID="login-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="password"
          error={passwordError}
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureTextEntry
          testID="login-password"
          textContentType="password"
          value={password}
        />
        {login.isError ? (
          <Text accessibilityLiveRegion="polite" tone="danger">
            {errorMessage(login.error)}
          </Text>
        ) : null}
        <Button label="Se connecter" loading={login.isPending} onPress={submit} testID="login-submit" />
        <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() || undefined })} variant="ghost" />
      </View>
    </Screen>
  );
}
