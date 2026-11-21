import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Button, Header, Screen, showToast, Text, TextField } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/format";

type Props = NativeStackScreenProps<RootStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const requestCode = async () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim().toLowerCase());
      setStep("code");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const reset = async () => {
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Le code contient 6 chiffres.");
      return;
    }
    const pwd = passwordProblem(password);
    if (pwd) {
      setError(pwd);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim().toLowerCase(), code.trim(), password);
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Login");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen
      scroll
      keyboard
      footer={
        step === "email" ? (
          <Button label="Recevoir un code" size="lg" loading={loading} onPress={requestCode} />
        ) : (
          <>
            <Button label="Changer mon mot de passe" size="lg" loading={loading} onPress={reset} />
            <Button label="Renvoyer un code" variant="ghost" onPress={requestCode} />
          </>
        )
      }
    >
      <Header back />
      <Text variant="title" accessibilityRole="header">
        Récupérer mon compte
      </Text>
      {step === "email" ? (
        <View style={styles.form}>
          <Text tone="muted">Indiquez votre e-mail : si un compte existe, vous recevrez un code à 6 chiffres valable 15 minutes.</Text>
          <TextField
            label="E-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            autoComplete="email"
            error={error}
          />
        </View>
      ) : (
        <View style={styles.form}>
          <Text tone="muted">
            Saisissez le code reçu par e-mail et choisissez un nouveau mot de passe. Vos autres sessions seront déconnectées.
          </Text>
          <TextField
            label="Code"
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
            secureToggle
            autoComplete="new-password"
            error={error}
          />
        </View>
      )}
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  form: { gap: t.spacing.lg }
}));
