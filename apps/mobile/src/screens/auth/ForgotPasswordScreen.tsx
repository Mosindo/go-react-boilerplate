import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Screen, Text, TextField, useFeedback } from "../../ui";
import { looksLikeEmail, passwordProblem } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const { toast } = useFeedback();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"email" | "code">("email");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => setStep("code")
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(email.trim(), code.trim(), password),
    onSuccess: () => {
      toast("Mot de passe modifié. Tu peux te connecter.", "success");
      navigation.replace("Login");
    }
  });

  if (step === "email") {
    return (
      <Screen
        footer={
          <Button
            label="Recevoir un code"
            loading={request.isPending}
            disabled={!looksLikeEmail(email)}
            onPress={() => request.mutate()}
          />
        }
      >
        <View style={{ gap: 6 }}>
          <Text variant="title">Mot de passe oublié</Text>
          <Text tone="muted">Indique ton e-mail : nous t’enverrons un code à 6 chiffres valable 15 minutes.</Text>
        </View>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
        />
        {request.isError ? <Text tone="danger">{errorMessage(request.error)}</Text> : null}
      </Screen>
    );
  }

  const pwdError = password ? passwordProblem(password) : null;
  return (
    <Screen
      footer={
        <Button
          label="Changer le mot de passe"
          loading={reset.isPending}
          disabled={!/^\d{6}$/.test(code.trim()) || !!passwordProblem(password)}
          onPress={() => reset.mutate()}
        />
      }
    >
      <View style={{ gap: 6 }}>
        <Text variant="title">Vérifie tes e-mails</Text>
        <Text tone="muted">Si un compte existe pour {email.trim()}, un code vient d’être envoyé.</Text>
      </View>
      <TextField
        label="Code reçu"
        value={code}
        onChangeText={setCode}
        keyboardType="number-pad"
        maxLength={6}
        autoComplete="one-time-code"
      />
      <TextField
        label="Nouveau mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        error={pwdError}
      />
      {reset.isError ? <Text tone="danger">{errorMessage(reset.error)}</Text> : null}
      <Button label="Renvoyer un code" variant="ghost" size="md" onPress={() => request.mutate()} />
    </Screen>
  );
}
