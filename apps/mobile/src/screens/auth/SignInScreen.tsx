import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useRef, useState } from "react";
import { StyleSheet, Text, View, type TextInput } from "react-native";
import * as authApi from "../../api/auth";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import type { RootStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { normalizeEmail, validateEmail } from "../../lib/validation";

type Props = NativeStackScreenProps<RootStackParamList, "SignIn">;

export default function SignInScreen({ navigation }: Props) {
  const theme = useTheme();
  const { startSession } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const passwordRef = useRef<TextInput>(null);

  const submit = useAsyncAction(async () => {
    const problem = validateEmail(email);
    setEmailError(problem);
    if (problem || !password) {
      throw new Error(problem ?? "Entrez votre mot de passe.");
    }
    await startSession(await authApi.login(normalizeEmail(email), password));
  });

  return (
    <Screen scroll>
      <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
        Content de vous revoir
      </Text>
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          keyboardType="email-address"
          label="Adresse e-mail"
          onChangeText={setEmail}
          onSubmitEditing={() => passwordRef.current?.focus()}
          returnKeyType="next"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="current-password"
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={() => void submit.run()}
          ref={passwordRef}
          returnKeyType="go"
          revealable
          secureTextEntry
          textContentType="password"
          value={password}
        />
        {submit.error ? (
          <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
        ) : null}
        <Button
          loading={submit.loading}
          onPress={() => void submit.run()}
          testID="signin-submit"
          title="Se connecter"
        />
        <Button
          onPress={() => navigation.navigate("ForgotPassword")}
          style={styles.link}
          title="Mot de passe oublié ?"
          variant="ghost"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { fontSize: 30, fontWeight: "700", letterSpacing: -0.5, marginVertical: spacing.xl },
  form: { marginTop: spacing.md },
  error: { fontSize: fontSizes.sm, marginBottom: spacing.md },
  link: { marginTop: spacing.sm }
});
