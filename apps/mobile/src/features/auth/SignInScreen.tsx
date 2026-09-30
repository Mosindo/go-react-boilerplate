import React, { useRef, useState } from "react";
import { StyleSheet, View, type TextInput } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text, TextField } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { EMAIL_PATTERN } from "../../lib/format";
import { useSession } from "../../lib/session/SessionProvider";
import type { AuthStackParamList } from "../../navigation/types";

type Props = NativeStackScreenProps<AuthStackParamList, "SignIn">;

export default function SignInScreen({ navigation }: Props) {
  const { signIn, expiredNotice } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    if (!EMAIL_PATTERN.test(email.trim()) || !password) {
      setError("Renseignez votre email et votre mot de passe.");
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
    <Screen scroll testID="signin-screen">
      <Text variant="title">Bon retour</Text>
      {expiredNotice ? <Text tone="muted">Votre session a expiré, reconnectez-vous.</Text> : null}
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        inputMode="email"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="signin-email"
        textContentType="emailAddress"
        value={email}
      />
      <TextField
        autoComplete="current-password"
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        ref={passwordRef}
        returnKeyType="go"
        secureTextEntry
        testID="signin-password"
        textContentType="password"
        value={password}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" testID="signin-error" tone="danger">
          {error}
        </Text>
      ) : null}
      <View style={styles.actions}>
        <Button fullWidth label="Se connecter" loading={loading} onPress={submit} testID="signin-submit" />
        <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("ForgotPassword", { email })} variant="ghost" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  actions: { gap: 8, marginTop: 8 }
});
