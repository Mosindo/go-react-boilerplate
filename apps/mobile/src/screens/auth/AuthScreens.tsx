import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { StyleSheet, View } from "react-native";

import { ApiError } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../auth/AuthContext";
import { isValidEmail, passwordError } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { useFeedback } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { errorMessage } from "../../ui/States";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";

type Props<K extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, K>;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen scroll contentStyle={styles.welcome}>
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Ionicons name="heart" size={44} color={colors.onPrimary} />
        </View>
        <Text variant="display" center>
          Alba
        </Text>
        <Text tone="muted" center>
          Des rencontres sincères, simplement. Gratuit, sans abonnement, sans limite de likes.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button
          label="Créer un compte"
          onPress={() => navigation.navigate("Register")}
          testID="welcome-register"
        />
        <Button
          label="J'ai déjà un compte"
          variant="secondary"
          onPress={() => navigation.navigate("Login")}
          testID="welcome-login"
        />
        <Text variant="caption" tone="muted" center>
          Réservé aux personnes majeures. En continuant, vous vous engagez à respecter les autres.
        </Text>
      </View>
    </Screen>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!isValidEmail(email) || !password) {
      setError("Saisissez votre e-mail et votre mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await login(email.trim(), password);
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? "E-mail ou mot de passe incorrect."
          : errorMessage(err),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Bon retour</Text>
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
        secure
        autoComplete="current-password"
        textContentType="password"
        onSubmitEditing={submit}
        testID="login-password"
      />
      {error ? (
        <Text tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : null}
      <Button label="Se connecter" onPress={submit} loading={busy} testID="login-submit" />
      <Button
        label="Mot de passe oublié ?"
        variant="ghost"
        onPress={() => navigation.navigate("ForgotPassword")}
      />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { register } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});

  const submit = async () => {
    const next: typeof errors = {};
    if (!isValidEmail(email)) next.email = "Adresse e-mail invalide.";
    const pwd = passwordError(password);
    if (pwd) next.password = pwd;
    setErrors(next);
    if (next.email || next.password) return;

    setBusy(true);
    try {
      await register(email.trim(), password);
    } catch (err) {
      if (err instanceof ApiError && err.code === "email_exists")
        setErrors({ email: "Un compte existe déjà avec cet e-mail." });
      else setErrors({ form: errorMessage(err) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Créer votre compte</Text>
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
        testID="register-email"
      />
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        hint="8 caractères minimum."
        secure
        autoComplete="new-password"
        textContentType="newPassword"
        onSubmitEditing={submit}
        testID="register-password"
      />
      {errors.form ? <Text tone="danger">{errors.form}</Text> : null}
      <Button label="Continuer" onPress={submit} loading={busy} testID="register-submit" />
      <Button
        label="J'ai déjà un compte"
        variant="ghost"
        onPress={() => navigation.navigate("Login")}
      />
    </Screen>
  );
}

export function ForgotPasswordScreen({ navigation }: Props<"ForgotPassword">) {
  const { toast } = useFeedback();
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      toast("Si un compte existe, un code vient d'être envoyé.", "success");
      navigation.replace("ResetPassword", { email: email.trim() });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Mot de passe oublié</Text>
      <Text tone="muted">Nous vous envoyons un code de réinitialisation valable 30 minutes.</Text>
      <TextField
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        error={error}
        autoCapitalize="none"
        keyboardType="email-address"
        autoComplete="email"
      />
      <Button label="Envoyer le code" onPress={submit} loading={busy} />
    </Screen>
  );
}

export function ResetPasswordScreen({ navigation, route }: Props<"ResetPassword">) {
  const { toast } = useFeedback();
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    const pwd = passwordError(password);
    if (!code.trim() || pwd) {
      setError(pwd ?? "Saisissez le code reçu par e-mail.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(route.params.email, code.trim(), password);
      toast("Mot de passe modifié. Connectez-vous.", "success");
      navigation.popToTop();
      navigation.navigate("Login");
    } catch (err) {
      setError(
        err instanceof ApiError && err.code === "invalid_code"
          ? "Code invalide ou expiré."
          : errorMessage(err),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Nouveau mot de passe</Text>
      <Text tone="muted">Entrez le code envoyé à {route.params.email}.</Text>
      <TextField
        label="Code"
        value={code}
        onChangeText={setCode}
        autoCapitalize="characters"
        autoCorrect={false}
      />
      <TextField
        label="Nouveau mot de passe"
        value={password}
        onChangeText={setPassword}
        secure
        autoComplete="new-password"
      />
      {error ? <Text tone="danger">{error}</Text> : null}
      <Button label="Réinitialiser" onPress={submit} loading={busy} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  welcome: { justifyContent: "space-between", flexGrow: 1 },
  hero: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.lg,
    paddingVertical: spacing.xxl,
  },
  logo: { width: 88, height: 88, borderRadius: 44, alignItems: "center", justifyContent: "center" },
  actions: { gap: spacing.md },
});
