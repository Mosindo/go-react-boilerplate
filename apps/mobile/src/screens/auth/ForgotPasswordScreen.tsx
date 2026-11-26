import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { requestPasswordReset, resetPassword } from "../../api/auth";
import { errorMessage } from "../../api/client";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"email" | "code">("email");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const sendCode = async () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await requestPasswordReset(email.trim());
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const submitReset = async () => {
    const problem = passwordProblem(password);
    if (code.trim().length < 6 || problem) {
      setError(problem ?? "Saisissez le code reçu par e-mail.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await resetPassword(email.trim(), code, password);
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.replace("Login");
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen
      edges={["top", "bottom"]}
      footer={
        step === "email" ? (
          <Button label="Recevoir un code" loading={loading} onPress={sendCode} testID="forgot-send" />
        ) : (
          <Button label="Changer mon mot de passe" loading={loading} onPress={submitReset} testID="forgot-reset" />
        )
      }
      scroll
    >
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Mot de passe oublié</Text>
        {step === "email" ? (
          <>
            <Text tone="muted">Indiquez votre e-mail : si un compte existe, vous recevrez un code valable 30 minutes.</Text>
            <TextField
              autoCapitalize="none"
              inputMode="email"
              keyboardType="email-address"
              label="E-mail"
              onChangeText={setEmail}
              testID="forgot-email"
              value={email}
            />
          </>
        ) : (
          <>
            <Text tone="muted">Si un compte existe pour {email.trim()}, un code vient de lui être envoyé.</Text>
            <TextField
              autoCapitalize="characters"
              autoCorrect={false}
              label="Code reçu"
              onChangeText={setCode}
              placeholder="ABCD234EFG"
              testID="forgot-code"
              value={code}
            />
            <TextField
              autoComplete="new-password"
              hint="8 caractères minimum, dont une lettre et un chiffre."
              label="Nouveau mot de passe"
              onChangeText={setPassword}
              secureToggle
              testID="forgot-password"
              value={password}
            />
            <Button label="Renvoyer un code" onPress={sendCode} size="md" style={styles.link} variant="ghost" />
          </>
        )}
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingTop: spacing.md },
  link: { alignSelf: "flex-start", paddingHorizontal: 0 }
});
