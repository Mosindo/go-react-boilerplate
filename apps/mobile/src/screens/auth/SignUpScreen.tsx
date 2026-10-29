import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useRef, useState } from "react";
import { StyleSheet, Text, View, type TextInput } from "react-native";
import * as authApi from "../../api/auth";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import type { RootStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "SignUp">;

export default function SignUpScreen({ navigation }: Props) {
  const theme = useTheme();
  const { startSession } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const passwordRef = useRef<TextInput>(null);

  const submit = useAsyncAction(async () => {
    const emailProblem = validateEmail(email);
    const passwordProblem = validatePassword(password);
    setEmailError(emailProblem);
    setPasswordError(passwordProblem);
    if (emailProblem || passwordProblem) {
      return;
    }
    await startSession(await authApi.register(normalizeEmail(email), password));
  });

  return (
    <Screen scroll>
      <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
        Créons votre compte
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
          autoComplete="new-password"
          error={passwordError}
          hint="8 caractères minimum."
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={() => void submit.run()}
          ref={passwordRef}
          returnKeyType="go"
          revealable
          secureTextEntry
          textContentType="newPassword"
          value={password}
        />
        {submit.error ? (
          <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
        ) : null}
        <Button
          loading={submit.loading}
          onPress={() => void submit.run()}
          testID="signup-submit"
          title="Continuer"
        />
        <Text style={[styles.legal, { color: theme.textMuted }]}>
          En continuant, vous confirmez avoir 18 ans ou plus. Sillage est réservé aux adultes.
        </Text>
        <Button
          onPress={() => navigation.navigate("SignIn")}
          style={styles.link}
          title="J'ai déjà un compte"
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
  legal: { fontSize: fontSizes.xs, lineHeight: 18, marginTop: spacing.lg, textAlign: "center" },
  link: { marginTop: spacing.sm }
});
