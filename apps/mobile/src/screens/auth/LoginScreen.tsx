import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthProvider";
import { errorMessage } from "../../lib/errors";
import { Button, Input, Screen, Text } from "../../ui";
import { spacing } from "../../theme/tokens";
import type { AuthStackParams } from "../../navigation/types";

export default function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!email.trim() || !password) {
      setError("Saisissez votre e-mail et votre mot de passe.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="login-screen">
      <Text variant="title">Content de vous revoir</Text>
      <View style={{ gap: spacing.lg }}>
        <Input
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="auth-email-input"
        />
        <Input
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="current-password"
          textContentType="password"
          onSubmitEditing={submit}
          testID="auth-password-input"
        />
        {error ? (
          <Text tone="danger" accessibilityRole="alert" testID="auth-error">
            {error}
          </Text>
        ) : null}
        <Button label="Se connecter" onPress={submit} loading={loading} testID="auth-submit-button" />
        <Button
          label="Mot de passe oublié ?"
          variant="ghost"
          onPress={() => navigation.navigate("ForgotPassword", { email })}
        />
      </View>
    </Screen>
  );
}
