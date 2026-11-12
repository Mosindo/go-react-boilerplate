import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField } from "../../ui";

export default function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!email.trim() || !password) {
      setError("Renseignez votre email et votre mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(await authApi.login(email.trim(), password));
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll keyboard padded={false}>
      <Header title="" onBack={navigation.goBack} />
      <View style={styles.body}>
        <Text variant="title" serif accessibilityRole="header">
          Bon retour
        </Text>
        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="login-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="password"
          textContentType="password"
          onSubmitEditing={submit}
          testID="login-password"
        />
        <InlineError message={error} />
        <Button label="Se connecter" onPress={submit} loading={busy} testID="login-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() })} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { paddingHorizontal: spacing.lg, gap: spacing.md }
});
