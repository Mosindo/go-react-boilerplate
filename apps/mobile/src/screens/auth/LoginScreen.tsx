import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { login } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { isValidEmail } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!isValidEmail(email) || !password) {
      setError("Saisissez votre e-mail et votre mot de passe.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await signIn(await login(email.trim(), password));
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen
      edges={["top", "bottom"]}
      footer={<Button label="Se connecter" loading={loading} onPress={submit} testID="login-submit" />}
      scroll
      testID="login-screen"
    >
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Bon retour parmi nous</Text>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          placeholder="vous@exemple.fr"
          testID="login-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="current-password"
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureToggle
          testID="login-password"
          textContentType="password"
          value={password}
        />
        {error ? (
          <Text accessibilityLiveRegion="polite" testID="login-error" tone="danger">
            {error}
          </Text>
        ) : null}
        <Button
          label="Mot de passe oublié ?"
          onPress={() => navigation.navigate("ForgotPassword", { email })}
          size="md"
          style={styles.link}
          variant="ghost"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingTop: spacing.md },
  link: { alignSelf: "flex-start", paddingHorizontal: 0 }
});
