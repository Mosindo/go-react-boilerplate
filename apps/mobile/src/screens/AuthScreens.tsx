import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthContext";
import { errorMessage } from "../lib/errors";
import type { AuthStackParams } from "../navigation/types";
import { useTheme } from "../theme";
import { Button, Input, Screen, Text, useToast } from "../ui";

type Props<T extends keyof AuthStackParams> = NativeStackScreenProps<AuthStackParams, T>;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  const t = useTheme();
  return (
    <Screen edges={["top", "bottom"]} style={styles.welcome}>
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: t.primary }]}>
          <Ionicons name="flame" size={44} color={t.onPrimary} />
        </View>
        <Text variant="display" style={styles.center}>
          Ember
        </Text>
        <Text tone="muted" style={styles.center}>
          Des rencontres sincères, entièrement gratuites. Sans abonnement, sans limite, sans
          publicité.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} />
        <Button
          label="J'ai déjà un compte"
          variant="secondary"
          onPress={() => navigation.navigate("Login")}
        />
      </View>
    </Screen>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { signIn } = useAuth();
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setLoading(true);
    try {
      await signIn(email.trim(), password);
    } catch (err) {
      toast.show(errorMessage(err), "error");
      setLoading(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Bon retour</Text>
      <Input
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="username"
      />
      <Input
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="current-password"
        textContentType="password"
        onSubmitEditing={() => void submit()}
      />
      <Button
        label="Se connecter"
        loading={loading}
        disabled={!EMAIL_RE.test(email.trim()) || password.length === 0}
        onPress={() => void submit()}
      />
      <Button
        label="Mot de passe oublié ?"
        variant="ghost"
        onPress={() => navigation.navigate("Forgot")}
      />
    </Screen>
  );
}

export function RegisterScreen() {
  const { signUp } = useAuth();
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const emailOk = EMAIL_RE.test(email.trim());
  const passwordOk = password.length >= 8;

  const submit = async () => {
    setLoading(true);
    try {
      await signUp(email.trim(), password);
    } catch (err) {
      toast.show(errorMessage(err), "error");
      setLoading(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Créer votre compte</Text>
      <Text tone="muted">Vous devez avoir au moins 18 ans pour utiliser Ember.</Text>
      <Input
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="username"
        error={email.length > 3 && !emailOk ? "Adresse e-mail invalide." : null}
      />
      <Input
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="new-password"
        textContentType="newPassword"
        maxLength={72}
        hint="8 caractères minimum."
        error={password.length > 0 && !passwordOk ? "Trop court (8 caractères minimum)." : null}
      />
      <Button
        label="Continuer"
        loading={loading}
        disabled={!emailOk || !passwordOk}
        onPress={() => void submit()}
      />
    </Screen>
  );
}

export function ForgotScreen({ navigation }: Props<"Forgot">) {
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setLoading(true);
    try {
      await authApi.forgotPassword(email.trim());
      toast.show("Si un compte existe, un e-mail vient d'être envoyé.");
      navigation.replace("Reset", { email: email.trim() });
    } catch (err) {
      toast.show(errorMessage(err), "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Mot de passe oublié</Text>
      <Text tone="muted">Nous vous enverrons un code de réinitialisation valable 1 heure.</Text>
      <Input
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        keyboardType="email-address"
        autoComplete="email"
      />
      <Button
        label="Envoyer le code"
        loading={loading}
        disabled={!EMAIL_RE.test(email.trim())}
        onPress={() => void submit()}
      />
      <Button
        label="J'ai déjà un code"
        variant="ghost"
        onPress={() => navigation.navigate("Reset")}
      />
    </Screen>
  );
}

export function ResetScreen({ navigation }: Props<"Reset">) {
  const toast = useToast();
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setLoading(true);
    try {
      await authApi.resetPassword(token.trim(), password);
      toast.show("Mot de passe modifié. Connectez-vous.");
      navigation.popToTop();
      navigation.navigate("Login");
    } catch (err) {
      toast.show(errorMessage(err), "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title">Nouveau mot de passe</Text>
      <Input
        label="Code reçu par e-mail"
        value={token}
        onChangeText={setToken}
        autoCapitalize="none"
        autoCorrect={false}
      />
      <Input
        label="Nouveau mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        maxLength={72}
        autoComplete="new-password"
        hint="8 caractères minimum."
      />
      <Button
        label="Réinitialiser"
        loading={loading}
        disabled={token.trim().length < 10 || password.length < 8}
        onPress={() => void submit()}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  welcome: { justifyContent: "space-between", padding: 24 },
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: 14 },
  logo: { width: 84, height: 84, borderRadius: 28, alignItems: "center", justifyContent: "center" },
  center: { textAlign: "center" },
  actions: { gap: 10 }
});
