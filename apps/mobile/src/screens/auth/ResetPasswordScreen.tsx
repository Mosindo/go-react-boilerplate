import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { StyleSheet, Text } from "react-native";
import * as authApi from "../../api/auth";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import type { RootStackParamList } from "../../navigation/types";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "ResetPassword">;

export default function ResetPasswordScreen({ navigation, route }: Props) {
  const theme = useTheme();
  const toast = useToast();
  const [email, setEmail] = useState(route.params.email);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [fieldError, setFieldError] = useState<{
    email?: string;
    password?: string;
    code?: string;
  }>({});

  const submit = useAsyncAction(async () => {
    const errors = {
      email: validateEmail(email) ?? undefined,
      code: code.trim() ? undefined : "Entrez le code reçu par e-mail.",
      password: validatePassword(password) ?? undefined
    };
    setFieldError(errors);
    if (errors.email || errors.code || errors.password) {
      return;
    }
    await authApi.resetPassword(normalizeEmail(email), code, password);
    toast.show("Mot de passe modifié. Connectez-vous.", "success");
    navigation.navigate("SignIn");
  });

  return (
    <Screen scroll>
      <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
        Nouveau mot de passe
      </Text>
      <TextField
        autoCapitalize="none"
        error={fieldError.email}
        keyboardType="email-address"
        label="Adresse e-mail"
        onChangeText={setEmail}
        value={email}
      />
      <TextField
        autoCapitalize="characters"
        autoCorrect={false}
        error={fieldError.code}
        hint="8 caractères, reçus par e-mail."
        label="Code de réinitialisation"
        onChangeText={setCode}
        value={code}
      />
      <TextField
        error={fieldError.password}
        label="Nouveau mot de passe"
        onChangeText={setPassword}
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
        title="Modifier le mot de passe"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { fontSize: 30, fontWeight: "700", letterSpacing: -0.5, marginVertical: spacing.xl },
  error: { fontSize: fontSizes.sm, marginBottom: spacing.md }
});
