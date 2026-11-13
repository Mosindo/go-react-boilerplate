import { useMutation } from "@tanstack/react-query";
import React, { useState } from "react";
import { View } from "react-native";
import { authApi } from "../../api/auth";
import type { RootScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Header, Screen, Text, TextField, useToast } from "../../ui";
import { describeError } from "../../utils/errors";

export default function ForgotPasswordScreen({ navigation, route }: RootScreenProps<"ForgotPassword">) {
  const { spacing } = useTheme();
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [step, setStep] = useState<"email" | "code">("email");

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => setStep("code")
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(email.trim(), code.trim(), password),
    onSuccess: () => {
      toast.show("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.replace("Login");
    }
  });

  return (
    <Screen scroll testID="forgot-screen">
      <Header back />
      <View style={{ gap: spacing.lg, marginTop: spacing.md }}>
        <Text variant="title" accessibilityRole="header">
          Récupérer votre compte
        </Text>
        {step === "email" ? (
          <>
            <Text tone="muted">Nous vous enverrons un code à 6 chiffres, valable 15 minutes.</Text>
            <TextField
              label="E-mail"
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              autoComplete="email"
              testID="forgot-email"
            />
            {request.error ? <Text tone="danger">{describeError(request.error)}</Text> : null}
            <Button
              label="Envoyer le code"
              onPress={() => request.mutate()}
              loading={request.isPending}
              disabled={email.trim().length < 5}
              testID="forgot-send"
            />
          </>
        ) : (
          <>
            <Text tone="muted">
              Si un compte existe pour {email.trim()}, un code vient d’être envoyé. Vérifiez aussi vos spams.
            </Text>
            <TextField
              label="Code reçu"
              value={code}
              onChangeText={(v) => setCode(v.replace(/\D/g, "").slice(0, 6))}
              keyboardType="number-pad"
              autoComplete="one-time-code"
              textContentType="oneTimeCode"
              testID="forgot-code"
            />
            <TextField
              label="Nouveau mot de passe"
              value={password}
              onChangeText={setPassword}
              secureToggle
              autoComplete="new-password"
              hint="8 caractères minimum."
              testID="forgot-password"
            />
            {reset.error ? <Text tone="danger">{describeError(reset.error)}</Text> : null}
            <Button
              label="Changer le mot de passe"
              onPress={() => reset.mutate()}
              loading={reset.isPending}
              disabled={code.length !== 6 || password.length < 8}
              testID="forgot-submit"
            />
            <Button label="Renvoyer un code" variant="ghost" size="md" onPress={() => request.mutate()} />
          </>
        )}
      </View>
    </Screen>
  );
}
