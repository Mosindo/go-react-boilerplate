import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { passwordProblem } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { useAuth } from "../../providers/AuthProvider";
import { useTheme } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "ResetPassword">;

export function ResetPasswordScreen({ route }: Props) {
  const { signIn } = useAuth();
  const { spacing } = useTheme();
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(route.params.email, code.trim(), password),
    onSuccess: signIn
  });
  const codeError = submitted && code.replace(/[\s-]/g, "").length !== 10 ? "Le code contient 10 caractères." : null;
  const passwordError = submitted ? passwordProblem(password) : null;

  return (
    <Screen keyboard scroll header={<Header back title="Nouveau mot de passe" />}>
      <Text tone="muted">Si un compte existe pour {route.params.email}, un code vient d’y être envoyé. Pense à vérifier tes indésirables.</Text>
      <View style={{ gap: spacing.md }}>
        <TextField autoCapitalize="characters" autoCorrect={false} error={codeError} label="Code reçu" maxLength={14} onChangeText={setCode} value={code} />
        <TextField
          error={passwordError}
          label="Nouveau mot de passe"
          onChangeText={setPassword}
          secureTextEntry
          textContentType="newPassword"
          value={password}
        />
        {reset.isError ? <Text tone="danger">{errorMessage(reset.error)}</Text> : null}
        <Button
          label="Changer mon mot de passe"
          loading={reset.isPending}
          onPress={() => {
            setSubmitted(true);
            if (code.replace(/[\s-]/g, "").length === 10 && !passwordProblem(password)) {
              reset.mutate();
            }
          }}
        />
        <Text center tone="subtle" variant="caption">
          Par sécurité, tes autres appareils seront déconnectés.
        </Text>
      </View>
    </Screen>
  );
}
