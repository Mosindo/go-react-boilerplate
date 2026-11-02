import React, { useState } from "react";
import { Pressable, Text, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/endpoints";
import { apiErrorMessage } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import { AppText, Button, Screen, TextField } from "../../components/ui";
import { useToast } from "../../components/Toast";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";
import { spacing, useTheme } from "../../theme/theme";
import type { AuthStackParamList } from "../../navigation/types";

type Props<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen>
      <View style={{ flex: 1, justifyContent: "center", gap: spacing.lg }}>
        <Text style={{ fontSize: 56 }}>🤝</Text>
        <AppText variant="title">Tandem</AppText>
        <AppText muted>Des rencontres sincères, sans abonnement, sans boost, sans limite. Tout est gratuit, pour tout le monde.</AppText>
      </View>
      <View style={{ gap: spacing.md }}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        <Text style={{ color: colors.textMuted, textAlign: "center", fontSize: 12 }}>Réservé aux personnes de 18 ans et plus.</Text>
      </View>
    </Screen>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!isValidEmail(email)) return setError("Adresse e-mail invalide.");
    if (!password) return setError("Saisissez votre mot de passe.");
    setBusy(true);
    setError(null);
    try {
      await login(normalizeEmail(email), password);
    } catch (e) {
      setError(apiErrorMessage(e, "Connexion impossible."));
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <AppText variant="title">Bon retour</AppText>
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
        testID="login-email"
      />
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="password"
        textContentType="password"
        onSubmitEditing={submit}
        testID="login-password"
      />
      {error ? <AppText color="#B3261E">{error}</AppText> : null}
      <Button label="Se connecter" onPress={submit} loading={busy} testID="login-submit" />
      <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("Forgot")} />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { register } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const next: typeof errors = {};
    if (!isValidEmail(email)) next.email = "Adresse e-mail invalide.";
    const problem = passwordProblem(password);
    if (problem) next.password = problem;
    setErrors(next);
    if (next.email || next.password) return;
    setBusy(true);
    try {
      await register(normalizeEmail(email), password);
    } catch (e) {
      setErrors({ form: apiErrorMessage(e, "Inscription impossible.") });
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <AppText variant="title">Créer votre compte</AppText>
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        error={errors.email}
        testID="register-email"
      />
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="new-password"
        error={errors.password}
        hint="8 caractères minimum, lettres et chiffres."
        testID="register-password"
      />
      {errors.form ? <AppText color="#B3261E">{errors.form}</AppText> : null}
      <Button label="Continuer" onPress={submit} loading={busy} testID="register-submit" />
      <Pressable onPress={() => navigation.navigate("Login")} accessibilityRole="link">
        <AppText center muted>
          Déjà inscrit·e ? Se connecter
        </AppText>
      </Pressable>
    </Screen>
  );
}

export function ForgotScreen({ navigation }: Props<"Forgot">) {
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const send = async () => {
    if (!isValidEmail(email)) return setError("Adresse e-mail invalide.");
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(normalizeEmail(email));
      setSent(true);
    } catch (e) {
      setError(apiErrorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const problem = passwordProblem(password);
    if (!token.trim()) return setError("Collez le code reçu par e-mail.");
    if (problem) return setError(problem);
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(token.trim(), password);
      toast.show("Mot de passe modifié. Connectez-vous.", "success");
      navigation.navigate("Login");
    } catch (e) {
      setError(apiErrorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <AppText variant="title">Récupérer mon compte</AppText>
      {!sent ? (
        <>
          <AppText muted>Nous vous envoyons un code de réinitialisation valable une heure.</AppText>
          <TextField label="E-mail" value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" />
          <Button label="Envoyer le code" onPress={send} loading={busy} />
          <Button label="J'ai déjà un code" variant="ghost" onPress={() => setSent(true)} />
        </>
      ) : (
        <>
          <AppText muted>Si un compte existe pour cette adresse, un e-mail vient de partir. Saisissez le code reçu.</AppText>
          <TextField label="Code de réinitialisation" value={token} onChangeText={setToken} autoCapitalize="none" autoCorrect={false} />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            hint="8 caractères minimum, lettres et chiffres."
          />
          <Button label="Changer le mot de passe" onPress={reset} loading={busy} />
        </>
      )}
      {error ? <AppText color="#B3261E">{error}</AppText> : null}
    </Screen>
  );
}
