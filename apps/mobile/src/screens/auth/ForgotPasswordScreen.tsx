import React, { useState } from "react";
import { View } from "react-native";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/errors";
import { useFeedback } from "../../feedback";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Card, Header, Screen, Text, TextField } from "../../ui";
import { validateEmail, validatePassword } from "../../utils/validation";

export default function ForgotPasswordScreen({ navigation, route }: AuthScreenProps<"ForgotPassword">) {
  const { toast } = useFeedback();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const requestCode = async () => {
    const invalid = validateEmail(email);
    if (invalid) {
      setError(invalid);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    const weak = validatePassword(password);
    if (weak) {
      setError(weak);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      toast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Login");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen keyboard scroll testID="forgot-screen">
      <Header onBack={() => navigation.goBack()} title="Mot de passe oublié" />
      {step === "email" ? (
        <View style={{ gap: spacing.md }}>
          <Text color="textMuted">Indiquez votre e-mail : nous vous enverrons un code à 6 chiffres, valable 15 minutes.</Text>
          <TextField
            autoCapitalize="none"
            error={error}
            inputMode="email"
            keyboardType="email-address"
            label="E-mail"
            onChangeText={setEmail}
            testID="forgot-email"
            value={email}
          />
          <Button label="Recevoir un code" loading={loading} onPress={requestCode} testID="forgot-submit" />
        </View>
      ) : (
        <View style={{ gap: spacing.md }}>
          <Card>
            <Text color="textMuted">
              Si un compte existe pour {email.trim()}, un code vient d’être envoyé. Pensez à vérifier vos indésirables.
            </Text>
          </Card>
          <TextField
            inputMode="numeric"
            keyboardType="number-pad"
            label="Code reçu"
            maxLength={6}
            onChangeText={setCode}
            testID="reset-code"
            value={code}
          />
          <TextField
            error={error}
            hint="8 caractères minimum, avec une lettre et un chiffre."
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureToggle
            testID="reset-password"
            value={password}
          />
          <Button label="Changer mon mot de passe" loading={loading} onPress={reset} testID="reset-submit" />
          <Button label="Renvoyer un code" onPress={requestCode} variant="ghost" />
        </View>
      )}
    </Screen>
  );
}
