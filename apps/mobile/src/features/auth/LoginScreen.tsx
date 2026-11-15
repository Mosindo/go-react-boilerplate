import React, { useRef, useState } from "react";
import { StyleSheet, View, type TextInput } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import { isValidEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { useSession } from "../../session/SessionProvider";
import { spacing } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    if (!isValidEmail(email) || !password) {
      setError("Saisissez votre e-mail et votre mot de passe.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await signIn(await authApi.login(email.trim(), password));
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="login-screen" edges={["top", "bottom"]}>
      <Header onBack={() => navigation.goBack()} />
      <Text variant="title" style={styles.title} accessibilityRole="header">
        Bon retour
      </Text>
      <View style={styles.form}>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          returnKeyType="next"
          onSubmitEditing={() => passwordRef.current?.focus()}
          testID="login-email"
        />
        <TextField
          ref={passwordRef}
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="current-password"
          textContentType="password"
          returnKeyType="go"
          onSubmitEditing={submit}
          testID="login-password"
        />
        <InlineError message={error} />
        <Button label="Se connecter" onPress={submit} loading={loading} testID="login-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword", { email })} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { marginTop: spacing.md, marginBottom: spacing.xl },
  form: { gap: spacing.md }
});
