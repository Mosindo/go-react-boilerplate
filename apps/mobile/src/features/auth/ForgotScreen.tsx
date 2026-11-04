import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { View } from "react-native";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import type { AuthStackParams } from "../../navigation/types";
import { Button, Input, Text, spacing, useToast } from "../../shared/ui";
import { validateEmail, validatePassword } from "../../utils/validation";
import { AuthLayout } from "./AuthLayout";

export function ForgotScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Forgot">) {
  const toast = useToast();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const sendCode = async () => {
    const e = validateEmail(email);
    if (e) {
      setError(e);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.forgot(email.trim());
      setStep("code");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const pe = validatePassword(password);
    if (!/^\d{8}$/.test(code.trim())) {
      setError("Le code contient 8 chiffres");
      return;
    }
    if (pe) {
      setError(pe);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.reset(email.trim(), code.trim(), password);
      toast.show("Mot de passe modifié. Connectez-vous.", "success");
      navigation.navigate("Login");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <AuthLayout title="Mot de passe oublié" subtitle={step === "email" ? "Nous vous envoyons un code à 8 chiffres par e-mail." : "Saisissez le code reçu et choisissez un nouveau mot de passe."}>
      <View style={{ gap: spacing.md }}>
        {step === "email" ? (
          <>
            <Input label="E-mail" value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" testID="forgot-email" />
            {error ? <Text tone="danger">{error}</Text> : null}
            <Button title="Envoyer le code" onPress={sendCode} loading={busy} testID="forgot-send" />
          </>
        ) : (
          <>
            <Input label="Code à 8 chiffres" value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={8} testID="forgot-code" />
            <Input label="Nouveau mot de passe" value={password} onChangeText={setPassword} secureTextEntry testID="forgot-password" />
            {error ? <Text tone="danger" accessibilityLiveRegion="polite">{error}</Text> : null}
            <Button title="Modifier le mot de passe" onPress={reset} loading={busy} testID="forgot-reset" />
            <Button title="Renvoyer un code" variant="ghost" onPress={sendCode} />
          </>
        )}
        <Button title="Retour" variant="ghost" onPress={() => navigation.goBack()} />
      </View>
    </AuthLayout>
  );
}
