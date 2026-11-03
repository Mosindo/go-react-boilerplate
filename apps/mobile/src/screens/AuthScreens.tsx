import React, { useState } from "react";
import { Text, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { api } from "../api";
import { errorMessage } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import type { AuthStackParams } from "../navigation/types";
import { spacing, useTheme } from "../theme";
import { Body, Button, ErrorText, Screen, TextField, Title } from "../ui/components";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function Brand() {
  const theme = useTheme();
  return (
    <View style={{ alignItems: "center", gap: spacing.xs, marginVertical: spacing.xl }}>
      <Text style={{ color: theme.primary, fontSize: 44, fontWeight: "900", letterSpacing: -1 }}>Braise</Text>
      <Text style={{ color: theme.textMuted, fontSize: 15 }}>Des rencontres sincères. Gratuites, pour de vrai.</Text>
    </View>
  );
}

type Props<T extends keyof AuthStackParams> = NativeStackScreenProps<AuthStackParams, T>;

export function LoginScreen({ navigation }: Props<"Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!EMAIL_PATTERN.test(email.trim()) || !password) {
      setError("Saisissez votre e-mail et votre mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen>
      <Brand />
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        testID="login-email"
      />
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="current-password"
        testID="login-password"
        onSubmitEditing={() => void submit()}
      />
      <ErrorText message={error} />
      <Button title="Se connecter" onPress={() => void submit()} loading={busy} testID="login-submit" />
      <Button title="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword")} />
      <Button title="Créer un compte" variant="secondary" onPress={() => navigation.navigate("Register")} testID="goto-register" />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { signUp } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!EMAIL_PATTERN.test(email.trim())) {
      setError("Saisissez une adresse e-mail valide.");
      return;
    }
    if (password.length < 8) {
      setError("Le mot de passe doit contenir au moins 8 caractères.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signUp(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen>
      <Brand />
      <Title>Créer un compte</Title>
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        testID="register-email"
      />
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="new-password"
        hint="8 caractères minimum"
        testID="register-password"
        onSubmitEditing={() => void submit()}
      />
      <ErrorText message={error} />
      <Body muted>En continuant, vous confirmez avoir au moins 18 ans.</Body>
      <Button title="Continuer" onPress={() => void submit()} loading={busy} testID="register-submit" />
      <Button title="J'ai déjà un compte" variant="ghost" onPress={() => navigation.navigate("Login")} />
    </Screen>
  );
}

export function ForgotPasswordScreen({ navigation }: Props<"ForgotPassword">) {
  const [step, setStep] = useState<"request" | "reset" | "done">("request");
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (task: () => Promise<void>, next: "reset" | "done") => {
    setBusy(true);
    setError(null);
    try {
      await task();
      setStep(next);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen>
      <Title>Récupérer mon compte</Title>
      {step === "request" ? (
        <>
          <Body muted>Nous vous envoyons un code de réinitialisation par e-mail.</Body>
          <TextField
            label="E-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            testID="forgot-email"
          />
          <ErrorText message={error} />
          <Button
            title="Envoyer le code"
            loading={busy}
            onPress={() => void run(() => api.forgotPassword(email.trim()), "reset")}
            testID="forgot-submit"
          />
        </>
      ) : null}
      {step === "reset" ? (
        <>
          <Body muted>Si un compte existe pour cette adresse, un code vient de partir. Il est valable 1 heure.</Body>
          <TextField label="Code reçu" value={token} onChangeText={setToken} autoCapitalize="none" testID="reset-token" />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            hint="8 caractères minimum"
            testID="reset-password"
          />
          <ErrorText message={error} />
          <Button
            title="Changer le mot de passe"
            loading={busy}
            onPress={() => void run(() => api.resetPassword(token.trim(), password), "done")}
            testID="reset-submit"
          />
        </>
      ) : null}
      {step === "done" ? (
        <>
          <Body>Votre mot de passe a été modifié. Reconnectez-vous avec le nouveau.</Body>
          <Button title="Retour à la connexion" onPress={() => navigation.navigate("Login")} />
        </>
      ) : null}
    </Screen>
  );
}
