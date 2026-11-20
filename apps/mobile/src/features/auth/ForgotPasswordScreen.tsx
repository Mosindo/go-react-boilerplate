import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { isValidEmail, normalizeEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const { spacing } = useTheme();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [submitted, setSubmitted] = useState(false);
  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(normalizeEmail(email)),
    onSuccess: () => navigation.navigate("ResetPassword", { email: normalizeEmail(email) })
  });
  const error = submitted && !isValidEmail(email) ? "Saisis une adresse e-mail valide." : null;

  return (
    <Screen keyboard scroll header={<Header back title="Mot de passe oublié" />}>
      <Text tone="muted">Indique l’e-mail de ton compte : nous t’enverrons un code à 10 caractères, valable 30 minutes.</Text>
      <View style={{ gap: spacing.md }}>
        <TextField autoCapitalize="none" error={error} inputMode="email" keyboardType="email-address" label="E-mail" onChangeText={setEmail} value={email} />
        {request.isError ? <Text tone="danger">{errorMessage(request.error)}</Text> : null}
        <Button
          label="Recevoir un code"
          loading={request.isPending}
          onPress={() => {
            setSubmitted(true);
            if (isValidEmail(email)) {
              request.mutate();
            }
          }}
        />
      </View>
    </Screen>
  );
}
