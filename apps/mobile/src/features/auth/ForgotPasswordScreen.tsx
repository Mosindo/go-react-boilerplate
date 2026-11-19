import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField, useFeedback } from "../../ui";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const { toast } = useFeedback();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"email" | "code">("email");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const requestCode = useMutation({
    mutationFn: () => authApi.forgotPassword(normalizeEmail(email)),
    onSuccess: () => {
      setStep("code");
      setSubmitted(false);
    }
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(normalizeEmail(email), code, password),
    onSuccess: () => {
      toast("Mot de passe mis à jour. Connectez-vous.", "success");
      navigation.navigate("Login");
    }
  });

  const pwdProblem = passwordProblem(password);

  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Récupérer mon compte</Text>
        {step === "email" ? (
          <>
            <Text tone="muted">Nous vous enverrons un code à 6 chiffres valable 15 minutes.</Text>
            <TextField
              label="E-mail"
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              autoComplete="email"
              error={submitted && !isValidEmail(email) ? "Saisissez une adresse e-mail valide." : null}
            />
            {requestCode.error ? <Text tone="danger">{errorMessage(requestCode.error)}</Text> : null}
            <Button
              label="Recevoir un code"
              loading={requestCode.isPending}
              onPress={() => {
                setSubmitted(true);
                if (isValidEmail(email)) requestCode.mutate();
              }}
            />
          </>
        ) : (
          <>
            <Text tone="muted">
              Si un compte existe pour {normalizeEmail(email)}, un code vient d’être envoyé. Pensez à vérifier vos spams.
            </Text>
            <TextField
              label="Code reçu"
              value={code}
              onChangeText={(v) => setCode(v.replace(/\D/g, "").slice(0, 6))}
              keyboardType="number-pad"
              autoComplete="one-time-code"
              textContentType="oneTimeCode"
              error={submitted && code.length !== 6 ? "Le code contient 6 chiffres." : null}
            />
            <TextField
              label="Nouveau mot de passe"
              value={password}
              onChangeText={setPassword}
              secureToggle
              autoComplete="new-password"
              error={submitted ? pwdProblem : null}
            />
            {reset.error ? <Text tone="danger">{errorMessage(reset.error)}</Text> : null}
            <Button
              label="Changer le mot de passe"
              loading={reset.isPending}
              onPress={() => {
                setSubmitted(true);
                if (code.length === 6 && !pwdProblem) reset.mutate();
              }}
            />
            <Button label="Renvoyer un code" variant="ghost" onPress={() => requestCode.mutate()} loading={requestCode.isPending} />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, paddingTop: spacing.md }
});
