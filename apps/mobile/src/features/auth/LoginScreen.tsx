import React, { useRef, useState } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Screen, Text, TextField } from "../../ui";
import { isValidEmail } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const passwordRef = useRef<TextInput>(null);
  const login = useMutation({ mutationFn: () => authApi.login(email.trim(), password), onSuccess: signIn });

  const emailError = submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const submit = () => {
    setSubmitted(true);
    if (isValidEmail(email) && password) {
      login.mutate();
    }
  };

  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="" />
      <Text variant="title">Bon retour parmi nous</Text>
      <Text style={styles.lead} tone="muted">
        Connectez-vous pour retrouver vos matchs et vos conversations.
      </Text>
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          onSubmitEditing={() => passwordRef.current?.focus()}
          returnKeyType="next"
          testID="login-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          ref={passwordRef}
          autoComplete="current-password"
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          returnKeyType="go"
          secureToggle
          testID="login-password"
          textContentType="password"
          value={password}
        />
        <InlineError message={login.isError ? errorMessage(login.error) : null} />
        <Button label="Se connecter" loading={login.isPending} onPress={submit} size="lg" testID="login-submit" />
        <Button
          label="Mot de passe oublié ?"
          onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() || undefined })}
          variant="ghost"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  lead: { marginTop: spacing.xs, marginBottom: spacing.xl },
  form: { gap: spacing.md }
});
