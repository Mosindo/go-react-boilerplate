import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { isValidEmail, normalizeEmail } from "../../lib/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Saisissez une adresse e-mail valide." : null;
  const passwordError = submitted && !password ? "Saisissez votre mot de passe." : null;

  const submit = () => {
    setSubmitted(true);
    if (!isValidEmail(email) || !password) return;
    login.mutate();
  };

  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Bon retour</Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
          testID="login-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="current-password"
          textContentType="password"
          onSubmitEditing={submit}
          error={passwordError}
          testID="login-password"
        />
        {login.error ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="login-error">
            {errorMessage(login.error)}
          </Text>
        ) : null}
        <Button label="Se connecter" onPress={submit} loading={login.isPending} testID="login-submit" />
        <Button
          label="Mot de passe oublié ?"
          variant="ghost"
          onPress={() => navigation.navigate("ForgotPassword", { email: normalizeEmail(email) || undefined })}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, paddingTop: spacing.md }
});
