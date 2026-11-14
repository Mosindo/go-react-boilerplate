import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Screen, Text, TextField, useToast } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/validation";

export function ForgotPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "ForgotPassword">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [submitted, setSubmitted] = useState(false);
  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => navigation.navigate("ResetPassword", { email: email.trim() })
  });
  const submit = () => {
    setSubmitted(true);
    if (isValidEmail(email)) request.mutate();
  };
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="" />
      <Text variant="title">Mot de passe oublié</Text>
      <Text style={styles.lead} tone="muted">
        Indiquez votre e-mail : si un compte existe, vous recevrez un code à 6 chiffres valable 30 minutes.
      </Text>
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null}
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          onSubmitEditing={submit}
          value={email}
        />
        <InlineError message={request.isError ? errorMessage(request.error) : null} />
        <Button label="Recevoir un code" loading={request.isPending} onPress={submit} size="lg" />
      </View>
    </Screen>
  );
}

export function ResetPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "ResetPassword">) {
  const toast = useToast();
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(route.params.email, code.trim(), password),
    onSuccess: () => {
      toast({ title: "Mot de passe mis à jour", body: "Vous pouvez vous connecter.", tone: "success" });
      navigation.popToTop();
      navigation.navigate("Login");
    }
  });
  const problem = passwordProblem(password);
  const codeValid = /^\d{6}$/.test(code.trim());
  const submit = () => {
    setSubmitted(true);
    if (codeValid && !problem) reset.mutate();
  };
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="" />
      <Text variant="title">Nouveau mot de passe</Text>
      <Text style={styles.lead} tone="muted">
        Saisissez le code envoyé à {route.params.email}.
      </Text>
      <View style={styles.form}>
        <TextField
          autoComplete="one-time-code"
          error={submitted && !codeValid ? "Le code contient 6 chiffres." : null}
          keyboardType="number-pad"
          label="Code"
          maxLength={6}
          onChangeText={setCode}
          textContentType="oneTimeCode"
          value={code}
        />
        <TextField
          autoComplete="new-password"
          error={submitted ? problem : null}
          label="Nouveau mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureToggle
          value={password}
        />
        <InlineError message={reset.isError ? errorMessage(reset.error) : null} />
        <Button label="Mettre à jour" loading={reset.isPending} onPress={submit} size="lg" />
        <Button label="Renvoyer un code" onPress={() => navigation.goBack()} variant="ghost" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  lead: { marginTop: spacing.xs, marginBottom: spacing.xl },
  form: { gap: spacing.md }
});
