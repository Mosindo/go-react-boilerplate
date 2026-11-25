import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../api/client";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { normalizeEmail, validateEmail } from "../../utils/validation";

export function ForgotPasswordScreen({ navigation }: AuthScreenProps<"ForgotPassword">) {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(normalizeEmail(email)),
    onSuccess: () => navigation.navigate("ResetPassword", { email: normalizeEmail(email) })
  });

  const submit = () => {
    setSubmitted(true);
    if (!validateEmail(email)) {
      request.mutate();
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Mot de passe oublié</Text>
        <Text variant="small" color="textMuted">
          Indiquez l’e-mail de votre compte : nous vous enverrons un code de réinitialisation valable 30 minutes.
        </Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          error={submitted ? validateEmail(email) : null}
          onSubmitEditing={submit}
        />
        {request.error ? (
          <Text variant="small" color="danger">
            {errorMessage(request.error)}
          </Text>
        ) : null}
        <Button label="Envoyer le code" onPress={submit} loading={request.isPending} />
        <Button label="J'ai déjà un code" variant="ghost" onPress={() => navigation.navigate("ResetPassword", undefined)} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md,
    paddingTop: spacing.md
  }
});
