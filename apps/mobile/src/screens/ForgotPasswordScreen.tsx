import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { errorMessage } from "../api/client";
import { NavHeader } from "../components/NavHeader";
import { Screen } from "../components/Screen";
import type { AuthStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Button, Input, Notice, Text } from "../shared/ui";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [step, setStep] = useState<"request" | "reset">("request");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");

  const request = useMutation({
    mutationFn: () => requestPasswordReset(email.trim()),
    onSuccess: () => setStep("reset")
  });
  const reset = useMutation({
    mutationFn: () => resetPassword(email.trim(), code.trim(), password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Connectez-vous.", "success");
      navigation.navigate("Auth");
    }
  });

  return (
    <Screen
      header={<NavHeader onBack={() => navigation.goBack()} title="Mot de passe oublié" />}
      scroll
      testID="forgot-screen"
    >
      {step === "request" ? (
        <>
          <Text tone="muted">
            Saisissez votre adresse e-mail : nous vous envoyons un code à usage unique valable 30 minutes.
          </Text>
          {request.isError ? <Notice title={errorMessage(request.error)} tone="danger" /> : null}
          <Input
            autoCapitalize="none"
            inputMode="email"
            keyboardType="email-address"
            label="Adresse e-mail"
            onChangeText={setEmail}
            testID="forgot-email"
            value={email}
          />
          <Button
            disabled={!email.trim()}
            label="Envoyer le code"
            loading={request.isPending}
            onPress={() => request.mutate()}
            size="lg"
            testID="forgot-send"
          />
        </>
      ) : (
        <>
          <Notice
            description="Si cette adresse est enregistrée, un code vient de vous être envoyé."
            title="Vérifiez vos e-mails"
            tone="success"
          />
          {reset.isError ? <Notice title={errorMessage(reset.error)} tone="danger" /> : null}
          <Input
            autoCapitalize="characters"
            label="Code reçu"
            maxLength={8}
            onChangeText={setCode}
            testID="forgot-code"
            value={code}
          />
          <Input
            autoCapitalize="none"
            helperText="8 caractères minimum."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            testID="forgot-password"
            value={password}
          />
          <Button
            disabled={code.trim().length < 8 || password.length < 8}
            label="Changer le mot de passe"
            loading={reset.isPending}
            onPress={() => reset.mutate()}
            size="lg"
            testID="forgot-confirm"
          />
          <Button
            label="Renvoyer un code"
            loading={request.isPending}
            onPress={() => request.mutate()}
            variant="ghost"
          />
        </>
      )}
    </Screen>
  );
}
