import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { View } from "react-native";
import { errorMessage } from "../../api/client";
import { useSession } from "../../hooks/session";
import type { AuthStackParams } from "../../navigation/types";
import { Button, Input, Text, spacing } from "../../shared/ui";
import { validateEmail } from "../../utils/validation";
import { AuthLayout } from "./AuthLayout";

export function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Login">) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const emailError = validateEmail(email);
    if (emailError || !password) {
      setError(emailError ?? "Saisissez votre mot de passe");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(email, password);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <AuthLayout title="Bon retour" subtitle="Connectez-vous pour retrouver vos conversations.">
      <View style={{ gap: spacing.md }}>
        <Input label="E-mail" value={email} onChangeText={setEmail} autoCapitalize="none" autoComplete="email" keyboardType="email-address" textContentType="emailAddress" testID="login-email" />
        <Input label="Mot de passe" value={password} onChangeText={setPassword} secureTextEntry autoComplete="password" textContentType="password" testID="login-password" onSubmitEditing={submit} />
        {error ? <Text tone="danger" accessibilityLiveRegion="polite">{error}</Text> : null}
        <Button title="Se connecter" onPress={submit} loading={busy} testID="login-submit" />
        <Button title="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("Forgot")} testID="login-forgot" />
        <Button title="Créer un compte" variant="secondary" onPress={() => navigation.navigate("Register")} testID="login-register" />
      </View>
    </AuthLayout>
  );
}
