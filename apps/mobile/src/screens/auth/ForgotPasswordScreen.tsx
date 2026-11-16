import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { errorMessage } from "../../utils/errors";

export default function ForgotPasswordScreen({ navigation }: AuthScreenProps<"ForgotPassword">) {
  const [email, setEmail] = useState("");
  const send = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => navigation.navigate("ResetPassword", { email: email.trim() })
  });
  return (
    <Screen scroll keyboard>
      <Header title="Mot de passe oublié" onBack={navigation.goBack} large />
      <View style={styles.form}>
        <Text tone="muted">Indique ton email : si un compte existe, tu recevras un code à 6 chiffres valable 30 minutes.</Text>
        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          testID="forgot-email"
        />
        {send.error ? <Text tone="danger">{errorMessage(send.error)}</Text> : null}
        <Button label="Recevoir un code" onPress={() => send.mutate()} loading={send.isPending} disabled={!email.includes("@")} testID="forgot-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
