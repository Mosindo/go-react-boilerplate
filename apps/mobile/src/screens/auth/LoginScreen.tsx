import React, { useRef, useState } from "react";
import { TextInput, View } from "react-native";
import { errorMessage } from "../../api/errors";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Card, Header, Screen, Text, TextField } from "../../ui";

export default function LoginScreen({ navigation }: AuthScreenProps<"Login">) {
  const { signIn, sessionNotice, clearNotice } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    if (!email.trim() || !password) {
      setError("Renseignez votre e-mail et votre mot de passe.");
      return;
    }
    setLoading(true);
    setError(null);
    clearNotice();
    try {
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen keyboard scroll testID="login-screen">
      <Header onBack={() => navigation.goBack()} title="Bon retour" subtitle="Connectez-vous pour retrouver vos matchs." />
      {sessionNotice ? (
        <Card>
          <Text color="textMuted">{sessionNotice}</Text>
        </Card>
      ) : null}
      <View style={{ gap: spacing.md }}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
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
          error={error}
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          returnKeyType="go"
          secureToggle
          testID="login-password"
          textContentType="password"
          value={password}
        />
      </View>
      <Button label="Se connecter" loading={loading} onPress={submit} testID="login-submit" />
      <Button
        label="Mot de passe oublié ?"
        onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() })}
        testID="login-forgot"
        variant="ghost"
      />
    </Screen>
  );
}
