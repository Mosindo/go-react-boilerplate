import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { ApiError } from "../../api/client";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Input, Notice } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export default function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [codeSent, setCodeSent] = useState(false);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(normalizeEmail(email)),
    onSuccess: () => setCodeSent(true),
    onError: () => setError("Envoi impossible pour le moment. Réessayez.")
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(normalizeEmail(email), code.trim(), password),
    onSuccess: () => setDone(true),
    onError: (err) => setError(err instanceof ApiError && err.status === 400 ? "Code invalide ou expiré." : "Réinitialisation impossible.")
  });

  const sendCode = () => {
    const message = validateEmail(email);
    setError(message);
    if (!message) request.mutate();
  };
  const submitReset = () => {
    const message = !code.trim() ? "Saisissez le code reçu par e-mail." : validatePassword(password);
    setError(message);
    if (!message) reset.mutate();
  };

  if (done) {
    return (
      <AuthLayout title="Mot de passe modifié">
        <Notice description="Vos autres appareils ont été déconnectés par sécurité." title="Vous pouvez vous reconnecter." tone="success" />
        <Button label="Retour à la connexion" onPress={() => navigation.navigate("Login")} />
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      subtitle={codeSent ? "Saisissez le code reçu puis votre nouveau mot de passe." : "Nous vous enverrons un code par e-mail."}
      title="Récupérer mon compte"
    >
      <Input autoCapitalize="none" editable={!codeSent} keyboardType="email-address" label="E-mail" onChangeText={setEmail} value={email} />
      {codeSent ? (
        <>
          <Notice title="Si un compte existe pour cette adresse, un code vient d'être envoyé." tone="success" />
          <Input autoCapitalize="characters" label="Code" onChangeText={setCode} value={code} />
          <Input label="Nouveau mot de passe" onChangeText={setPassword} secureTextEntry value={password} />
        </>
      ) : null}
      {error ? <Notice title={error} tone="danger" /> : null}
      {codeSent ? (
        <Button label="Changer mon mot de passe" loading={reset.isPending} onPress={submitReset} />
      ) : (
        <Button label="Envoyer le code" loading={request.isPending} onPress={sendCode} />
      )}
      <Button label="Retour" onPress={() => navigation.goBack()} variant="ghost" />
    </AuthLayout>
  );
}
