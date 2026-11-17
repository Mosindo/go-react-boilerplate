import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import type { RootScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Screen, Text, TextField, showToast } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/format";

export function ForgotPasswordScreen({ navigation, route }: RootScreenProps<"ForgotPassword">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [submitted, setSubmitted] = useState(false);
  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => navigation.navigate("ResetPassword", { email: email.trim() })
  });

  function submit() {
    setSubmitted(true);
    if (isValidEmail(email)) request.mutate();
  }

  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={styles.form}>
        <Text tone="muted">Indique ton e-mail : si un compte existe, tu recevras un code à 8 caractères valable 30 minutes.</Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          error={submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null}
        />
        {request.isError ? <Text tone="danger">{errorMessage(request.error)}</Text> : null}
        <Button label="Recevoir un code" loading={request.isPending} onPress={submit} />
        <Button
          label="J'ai déjà un code"
          variant="ghost"
          onPress={() => isValidEmail(email) && navigation.navigate("ResetPassword", { email: email.trim() })}
        />
      </View>
    </Screen>
  );
}

export function ResetPasswordScreen({ navigation, route }: RootScreenProps<"ResetPassword">) {
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(route.params.email, code.trim(), password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Tu peux te connecter.", "success");
      navigation.popToTop();
      navigation.navigate("Login");
    }
  });

  const passwordError = submitted ? passwordProblem(password) : null;
  const codeError = submitted && code.trim().length !== 8 ? "Le code contient 8 caractères." : null;

  function submit() {
    setSubmitted(true);
    if (code.trim().length === 8 && !passwordProblem(password)) reset.mutate();
  }

  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={styles.form}>
        <Text tone="muted">Code envoyé à {route.params.email} (pense à vérifier tes spams).</Text>
        <TextField
          label="Code reçu"
          value={code}
          onChangeText={(v) => setCode(v.toUpperCase())}
          autoCapitalize="characters"
          autoCorrect={false}
          maxLength={10}
          error={codeError}
        />
        <TextField
          label="Nouveau mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          error={passwordError}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
        />
        {reset.isError ? <Text tone="danger">{errorMessage(reset.error)}</Text> : null}
        <Button label="Changer mon mot de passe" loading={reset.isPending} onPress={submit} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.lg,
    paddingTop: spacing.lg
  }
});
