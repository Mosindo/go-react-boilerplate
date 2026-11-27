import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordIssue } from "../../utils/format";
import type { AuthScreenProps } from "../../navigation/types";

export function ForgotPasswordScreen({ navigation }: AuthScreenProps<"ForgotPassword">) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim().toLowerCase()),
    onSuccess: () => {
      setTouched(false);
      setStep("code");
    }
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(email.trim().toLowerCase(), code.trim(), password),
    onSuccess: () => navigation.replace("Login", { notice: "Mot de passe modifié. Tu peux te connecter." })
  });

  return (
    <Screen scroll keyboard>
      <Header onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <Text variant="title" accessibilityRole="header">
          Récupérer ton compte
        </Text>
        {step === "email" ? (
          <>
            <Text tone="muted">Indique ton e-mail : si un compte existe, tu recevras un code valable 30 minutes.</Text>
            <TextField
              label="E-mail"
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              autoComplete="email"
              error={touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null}
            />
            <InlineError message={request.error ? errorMessage(request.error) : null} />
            <Button
              label="Recevoir un code"
              size="lg"
              loading={request.isPending}
              onPress={() => {
                setTouched(true);
                if (isValidEmail(email)) request.mutate();
              }}
            />
          </>
        ) : (
          <>
            <Text tone="muted">Saisis le code reçu par e-mail et choisis un nouveau mot de passe. Tous tes appareils seront déconnectés.</Text>
            <TextField label="Code" value={code} onChangeText={(v) => setCode(v.toUpperCase())} autoCapitalize="characters" maxLength={12} />
            <TextField
              label="Nouveau mot de passe"
              value={password}
              onChangeText={setPassword}
              secure
              autoComplete="new-password"
              error={touched ? passwordIssue(password) : null}
              hint="8 caractères minimum, avec au moins une lettre et un chiffre."
            />
            <InlineError message={reset.error ? errorMessage(reset.error) : null} />
            <Button
              label="Changer mon mot de passe"
              size="lg"
              loading={reset.isPending}
              onPress={() => {
                setTouched(true);
                if (code.trim().length >= 6 && !passwordIssue(password)) reset.mutate();
              }}
            />
            <Button label="Renvoyer un code" variant="ghost" onPress={() => request.mutate()} />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, maxWidth: 480, width: "100%", alignSelf: "center", paddingTop: spacing.md }
});
