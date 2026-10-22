import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { isValidEmail, normalizeEmail, passwordIssue } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";

type Mode = "login" | "register";

export default function AuthScreen() {
  const { signIn, signUp } = useAuth();
  const navigation = useNavigation<NativeStackNavigationProp<AuthStackParamList>>();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const emailError = email.length > 0 && !isValidEmail(email) ? "Adresse email invalide." : null;
  const passwordError = mode === "register" && password.length > 0 ? passwordIssue(password) : null;

  const submit = async () => {
    if (!isValidEmail(email)) {
      setError("Entrez une adresse email valide.");
      return;
    }
    const issue = mode === "register" ? passwordIssue(password) : password ? null : "Entrez votre mot de passe.";
    if (issue) {
      setError(issue);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      if (mode === "login") {
        await signIn(normalizeEmail(email), password);
      } else {
        await signUp(normalizeEmail(email), password);
      }
    } catch (err) {
      setError(errorMessage(err, "Connexion impossible."));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ScreenContainer contentMaxWidth={440} testID="auth-screen">
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <View style={styles.brand}>
            <Text tone="primary" variant="eyebrow" weight="bold">
              Aurore
            </Text>
            <Text accessibilityRole="header" variant="title" weight="bold">
              {mode === "login" ? "Content de vous revoir" : "Faisons connaissance"}
            </Text>
            <Text tone="muted">Des rencontres sincères, 100 % gratuites : pas d&apos;abonnement, pas de likes comptés.</Text>
          </View>

          <View style={styles.form}>
            <Input
              autoCapitalize="none"
              autoComplete="email"
              error={emailError}
              keyboardType="email-address"
              label="Email"
              onChangeText={setEmail}
              testID="auth-email"
              textContentType="emailAddress"
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
              textContentType={mode === "login" ? "password" : "newPassword"}
              value={password}
            />
            {error ? <Notice description={error} title="Connexion impossible" tone="danger" /> : null}
            <Button
              label={mode === "login" ? "Se connecter" : "Créer mon compte"}
              loading={submitting}
              onPress={submit}
              size="lg"
              testID="auth-submit"
            />
            {mode === "login" ? (
              <Button label="Mot de passe oublié ?" onPress={() => navigation.navigate("Forgot")} testID="auth-forgot" variant="ghost" />
            ) : null}
          </View>

          <Button
            label={mode === "login" ? "Pas encore de compte ? Créer un compte" : "Déjà inscrit ? Se connecter"}
            onPress={() => {
              setMode(mode === "login" ? "register" : "login");
              setError(null);
            }}
            testID="auth-switch"
            variant="outline"
          />
          {mode === "register" ? (
            <Text style={styles.legal} tone="muted" variant="caption">
              En créant un compte, vous confirmez avoir 18 ans ou plus.
            </Text>
          ) : null}
        </ScrollView>
      </KeyboardAvoidingView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { flexGrow: 1, justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.xl },
  brand: { gap: spacing.sm },
  form: { gap: spacing.lg },
  legal: { textAlign: "center" }
});
