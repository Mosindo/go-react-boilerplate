import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import { authApi } from "../api";
import { Button } from "../components/Button";
import { useFeedback } from "../components/Feedback";
import { Screen } from "../components/Screen";
import { Text } from "../components/Text";
import { TextField } from "../components/TextField";
import { validEmail, validPassword } from "../lib/format";
import type { AuthStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";

/** Two steps: request a code by email, then choose a new password. */
export function ForgotScreen() {
  const theme = useTheme();
  const navigation = useNavigation();
  const route = useRoute<RouteProp<AuthStackParamList, "Forgot">>();
  const { toast } = useFeedback();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"email" | "code">("email");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const requestCode = async () => {
    if (!validEmail(email)) return setError("Entrez une adresse email valide.");
    setLoading(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("code");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Envoi impossible.");
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    if (!/^\d{8}$/.test(code.trim())) return setError("Le code contient 8 chiffres.");
    const issue = validPassword(password);
    if (issue) return setError(issue);
    setLoading(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      toast("Mot de passe mis à jour. Connectez-vous.", "success");
      navigation.goBack();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Réinitialisation impossible.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll contentStyle={styles.content}>
      <Text variant="title">Récupérer mon compte</Text>
      {step === "email" ? (
        <View style={styles.form}>
          <Text muted>
            Entrez l&apos;email de votre compte : nous vous enverrons un code à 8 chiffres valable 30 minutes.
          </Text>
          <TextField
            label="Email"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            testID="forgot-email"
          />
          {error ? (
            <Text color={theme.danger} accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          <Button label="Envoyer le code" onPress={requestCode} loading={loading} testID="send-code" />
        </View>
      ) : (
        <View style={styles.form}>
          <Text muted>Si un compte existe pour {email.trim()}, un code vient d&apos;être envoyé.</Text>
          <TextField
            label="Code à 8 chiffres"
            value={code}
            onChangeText={setCode}
            keyboardType="number-pad"
            maxLength={8}
            testID="reset-code"
          />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secure
            autoCapitalize="none"
            testID="reset-password"
          />
          {error ? (
            <Text color={theme.danger} accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          <Button label="Changer le mot de passe" onPress={reset} loading={loading} testID="reset-submit" />
          <Button label="Renvoyer un code" variant="ghost" small onPress={requestCode} />
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 480, width: "100%", alignSelf: "center", gap: spacing.xl },
  form: { gap: spacing.lg }
});
