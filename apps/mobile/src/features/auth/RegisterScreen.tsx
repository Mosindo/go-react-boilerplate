import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { View } from "react-native";
import { errorMessage } from "../../api/client";
import { useSession } from "../../hooks/session";
import type { AuthStackParams } from "../../navigation/types";
import { Button, Input, Text, spacing } from "../../shared/ui";
import { validateEmail, validatePassword } from "../../utils/validation";
import { AuthLayout } from "./AuthLayout";

export function RegisterScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Register">) {
  const { signUp } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string | null; password?: string | null; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const next = { email: validateEmail(email), password: validatePassword(password) };
    setErrors(next);
    if (next.email || next.password) return;
    setBusy(true);
    try {
      await signUp(email, password);
    } catch (err) {
      setErrors({ form: errorMessage(err) });
      setBusy(false);
    }
  };

  return (
    <AuthLayout title="Créer votre compte" subtitle="Gratuit, sans abonnement, sans limite de likes.">
      <View style={{ gap: spacing.md }}>
        <Input label="E-mail" value={email} onChangeText={setEmail} error={errors.email} autoCapitalize="none" autoComplete="email" keyboardType="email-address" textContentType="emailAddress" testID="register-email" />
        <Input label="Mot de passe" value={password} onChangeText={setPassword} error={errors.password} hint="8 caractères minimum, avec une lettre et un chiffre" secureTextEntry textContentType="newPassword" testID="register-password" />
        <Text variant="caption" tone="muted">Liane est réservé aux personnes de 18 ans et plus. Votre âge sera vérifié à l'étape suivante.</Text>
        {errors.form ? <Text tone="danger" accessibilityLiveRegion="polite">{errors.form}</Text> : null}
        <Button title="Continuer" onPress={submit} loading={busy} testID="register-submit" />
        <Button title="J'ai déjà un compte" variant="ghost" onPress={() => navigation.goBack()} />
      </View>
    </AuthLayout>
  );
}
