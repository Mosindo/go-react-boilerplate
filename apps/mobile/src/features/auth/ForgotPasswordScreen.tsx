import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text, TextField } from "../../design/components";
import { authApi } from "../../lib/api/endpoints";
import { errorMessage } from "../../lib/api/client";
import { EMAIL_PATTERN, passwordProblem } from "../../lib/format";
import { showToast } from "../../lib/toast";
import type { AuthStackParamList } from "../../navigation/types";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation, route }: Props) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const requestCode = async () => {
    if (!EMAIL_PATTERN.test(email.trim())) {
      setError("Adresse email invalide.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    const problem = passwordProblem(password);
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("SignIn");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Text variant="title">Récupérer votre compte</Text>
      {step === "email" ? (
        <>
          <Text tone="muted">Indiquez votre email : si un compte existe, vous recevrez un code à 6 chiffres.</Text>
          <TextField
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            label="Email"
            onChangeText={setEmail}
            onSubmitEditing={requestCode}
            value={email}
          />
        </>
      ) : (
        <>
          <Text tone="muted">Si un compte est associé à {email.trim()}, un code vient d’être envoyé. Il est valable 30 minutes.</Text>
          <TextField
            autoComplete="one-time-code"
            keyboardType="number-pad"
            label="Code reçu"
            maxLength={6}
            onChangeText={setCode}
            textContentType="oneTimeCode"
            value={code}
          />
          <TextField
            autoComplete="new-password"
            hint="8 caractères minimum, avec au moins une lettre et un chiffre."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            onSubmitEditing={reset}
            secureTextEntry
            textContentType="newPassword"
            value={password}
          />
        </>
      )}
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger">
          {error}
        </Text>
      ) : null}
      <View style={styles.actions}>
        {step === "email" ? (
          <Button fullWidth label="Recevoir un code" loading={loading} onPress={requestCode} />
        ) : (
          <>
            <Button fullWidth label="Changer le mot de passe" loading={loading} onPress={reset} />
            <Button label="Renvoyer un code" onPress={requestCode} variant="ghost" />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  actions: { gap: 8, marginTop: 8 }
});
