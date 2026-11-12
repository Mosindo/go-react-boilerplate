import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField, showToast } from "../../ui";
import { validateEmail, validatePassword } from "../../utils/format";

export default function ForgotPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "ForgotPassword">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const request = async () => {
    const invalid = validateEmail(email);
    if (invalid) {
      setError(invalid);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setSent(true);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const invalid = !/^\d{6}$/.test(code) ? "Le code contient 6 chiffres." : validatePassword(password);
    if (invalid) {
      setError(invalid);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim(), code, password);
      showToast("Mot de passe mis à jour", "success");
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
          {sent ? "Vérifiez vos emails" : "Mot de passe oublié"}
        </Text>
        {!sent ? (
          <>
            <Text tone="muted">Indiquez l’email de votre compte : nous vous enverrons un code à 6 chiffres.</Text>
            <TextField label="Email" value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" autoComplete="email" />
            <InlineError message={error} />
            <Button label="Recevoir un code" onPress={request} loading={busy} />
          </>
        ) : (
          <>
            <Text tone="muted">Si un compte existe pour {email.trim()}, un code valable 15 minutes vient d’être envoyé.</Text>
            <TextField
              label="Code"
              value={code}
              onChangeText={(v) => setCode(v.replace(/\D/g, "").slice(0, 6))}
              keyboardType="number-pad"
              textContentType="oneTimeCode"
              autoComplete="one-time-code"
            />
            <TextField
              label="Nouveau mot de passe"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoComplete="new-password"
              hint="8 caractères minimum, avec une lettre et un chiffre."
            />
            <InlineError message={error} />
            <Button label="Changer le mot de passe" onPress={reset} loading={busy} />
            <Button label="Renvoyer un code" variant="ghost" onPress={request} />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { paddingHorizontal: spacing.lg, gap: spacing.md }
});
