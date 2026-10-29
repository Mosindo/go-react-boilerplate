import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { StyleSheet, Text } from "react-native";
import * as authApi from "../../api/auth";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { normalizeEmail, validateEmail } from "../../lib/validation";
import type { RootStackParamList } from "../../navigation/types";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation }: Props) {
  const theme = useTheme();
  const [email, setEmail] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);

  const submit = useAsyncAction(async () => {
    const problem = validateEmail(email);
    setEmailError(problem);
    if (problem) {
      return;
    }
    const normalized = normalizeEmail(email);
    await authApi.forgotPassword(normalized);
    navigation.navigate("ResetPassword", { email: normalized });
  });

  return (
    <Screen scroll>
      <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
        Mot de passe oublié
      </Text>
      <Text style={[styles.body, { color: theme.textMuted }]}>
        Entrez votre adresse e-mail : si un compte existe, nous vous enverrons un code de
        réinitialisation valable 30 minutes.
      </Text>
      <TextField
        autoCapitalize="none"
        error={emailError}
        keyboardType="email-address"
        label="Adresse e-mail"
        onChangeText={setEmail}
        onSubmitEditing={() => void submit.run()}
        returnKeyType="send"
        value={email}
      />
      {submit.error ? (
        <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
      ) : null}
      <Button loading={submit.loading} onPress={() => void submit.run()} title="Envoyer le code" />
      <Button
        onPress={() => navigation.navigate("ResetPassword", { email: normalizeEmail(email) })}
        style={styles.link}
        title="J'ai déjà un code"
        variant="ghost"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { fontSize: 30, fontWeight: "700", letterSpacing: -0.5, marginTop: spacing.xl },
  body: { fontSize: fontSizes.md, lineHeight: 22, marginVertical: spacing.lg },
  error: { fontSize: fontSizes.sm, marginBottom: spacing.md },
  link: { marginTop: spacing.sm }
});
