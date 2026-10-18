import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { APP_NAME } from "../components/labels";
import { Screen } from "../components/Screen";
import { useAuth, useLogin, useRegister } from "../hooks/useAuth";
import type { AuthStackParamList } from "../navigation/types";
import { Button, Input, Notice, Text, colors, spacing } from "../shared/ui";

type Mode = "login" | "register";
type Props = NativeStackScreenProps<AuthStackParamList, "Auth">;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function AuthScreen({ navigation }: Props) {
  const { notice } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmedAdult, setConfirmedAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const login = useLogin();
  const register = useRegister();
  const mutation = mode === "login" ? login : register;

  const emailError = submitted && !EMAIL_PATTERN.test(email.trim()) ? "Adresse e-mail invalide." : null;
  const passwordError = submitted && password.length < 8 ? "Au moins 8 caractères." : null;
  const adultError = submitted && mode === "register" && !confirmedAdult ? "Vous devez avoir 18 ans ou plus." : null;

  const submit = () => {
    setSubmitted(true);
    if (!EMAIL_PATTERN.test(email.trim()) || password.length < 8 || (mode === "register" && !confirmedAdult)) {
      return;
    }
    mutation.mutate({ email, password });
  };

  return (
    <Screen scroll testID="auth-screen">
      <View style={styles.brand}>
        <Text style={styles.logo} tone="primary" weight="bold">
          {APP_NAME}
        </Text>
        <Text style={styles.tagline} tone="muted">
          Des rencontres sincères. Gratuites, sans limite, sans abonnement.
        </Text>
      </View>

      <View style={styles.tabs}>
        <Button label="Connexion" onPress={() => setMode("login")} size="sm" style={styles.tab} testID="tab-login" variant={mode === "login" ? "primary" : "ghost"} />
        <Button label="Créer un compte" onPress={() => setMode("register")} size="sm" style={styles.tab} testID="tab-register" variant={mode === "register" ? "primary" : "ghost"} />
      </View>

      {notice ? <Notice title={notice} tone="warning" /> : null}
      {mutation.isError ? <Notice title={errorMessage(mutation.error)} tone="danger" /> : null}

      <Input
        autoCapitalize="none"
        autoComplete="email"
        error={emailError}
        inputMode="email"
        keyboardType="email-address"
        label="Adresse e-mail"
        onChangeText={setEmail}
        testID="auth-email"
        value={email}
      />
      <Input
        autoCapitalize="none"
        autoComplete={mode === "login" ? "current-password" : "new-password"}
        error={passwordError}
        helperText={mode === "register" ? "8 caractères minimum." : undefined}
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        testID="auth-password"
        value={password}
      />

      {mode === "register" ? (
        <View style={styles.adult}>
          <Button
            accessibilityRole="checkbox"
            accessibilityState={{ checked: confirmedAdult }}
            label={confirmedAdult ? "✓  J'ai 18 ans ou plus" : "J'ai 18 ans ou plus"}
            onPress={() => setConfirmedAdult((v) => !v)}
            size="sm"
            testID="auth-adult"
            variant={confirmedAdult ? "secondary" : "outline"}
          />
          {adultError ? (
            <Text tone="danger" variant="caption" weight="medium">
              {adultError}
            </Text>
          ) : null}
        </View>
      ) : null}

      <Button
        label={mode === "login" ? "Se connecter" : "Créer mon compte"}
        loading={mutation.isPending}
        onPress={submit}
        size="lg"
        testID="auth-submit"
      />

      {mode === "login" ? (
        <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("ForgotPassword", { email })} testID="auth-forgot" variant="ghost" />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  brand: { alignItems: "center", gap: spacing.sm, paddingVertical: spacing.xl },
  logo: { fontSize: 44, lineHeight: 52, letterSpacing: -1.5 },
  tagline: { textAlign: "center", maxWidth: 320 },
  tabs: { flexDirection: "row", gap: spacing.sm, backgroundColor: colors.surfaceSubtle, borderRadius: 24, padding: spacing.xxs },
  tab: { flex: 1 },
  adult: { gap: spacing.xs, alignItems: "flex-start" }
});
