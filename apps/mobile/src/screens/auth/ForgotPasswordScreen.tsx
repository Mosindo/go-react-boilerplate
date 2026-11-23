import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { requestPasswordReset, resetPassword } from "../../api/auth";
import type { AuthStackParamList } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Card, Input, Text } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [sent, setSent] = useState(false);

  const request = useMutation({
    mutationFn: () => requestPasswordReset(email.trim()),
    onSuccess: () => setSent(true)
  });
  const reset = useMutation({
    mutationFn: () => resetPassword(email.trim(), code.trim(), password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Tu peux te connecter.", "success");
      navigation.navigate("Login");
    }
  });

  return (
    <Screen testID="forgot-screen">
      <ScreenHeader subtitle="On t'envoie un code par email pour choisir un nouveau mot de passe." title="Récupérer mon compte" />
      <Input
        autoCapitalize="none"
        editable={!sent}
        inputMode="email"
        keyboardType="email-address"
        label="Email du compte"
        onChangeText={setEmail}
        value={email}
      />
      {!sent ? (
        <Button
          disabled={!/.+@.+\..+/.test(email.trim())}
          label="Recevoir un code"
          loading={request.isPending}
          onPress={() => request.mutate()}
        />
      ) : (
        <>
          <Card tone="primary">
            <Text variant="label">
              Si un compte existe pour cette adresse, un code à 8 caractères vient d&apos;être envoyé. Il expire dans 30 minutes.
            </Text>
          </Card>
          <Input autoCapitalize="characters" label="Code reçu" maxLength={12} onChangeText={setCode} value={code} />
          <Input
            error={reset.isError ? errorMessage(reset.error) : null}
            helperText="8 caractères minimum, avec une lettre et un chiffre."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            value={password}
          />
          <Button
            disabled={code.trim().length < 8 || password.length < 8}
            label="Changer le mot de passe"
            loading={reset.isPending}
            onPress={() => reset.mutate()}
          />
          <Button label="Renvoyer un code" onPress={() => request.mutate()} size="sm" variant="ghost" />
        </>
      )}
      {request.isError ? <Text tone="danger">{errorMessage(request.error)}</Text> : null}
    </Screen>
  );
}
