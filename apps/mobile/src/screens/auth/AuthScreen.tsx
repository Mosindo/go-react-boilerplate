import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { AppText, Banner, Button, Chip, Row, Screen, TextField } from "../../components/ui";
import { useAuth } from "../../auth/AuthContext";
import { errorMessage } from "../../api/http";
import { validateEmail, validatePassword } from "../../utils/validation";
import { spacing } from "../../theme/theme";
import type { AuthStackParams } from "../../navigation/types";

export default function AuthScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Auth">) {
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const emailError = submitted ? validateEmail(email) : null;
  const passwordError = submitted && mode === "register" ? validatePassword(password) : null;

  const submit = async () => {
    setSubmitted(true);
    setError(null);
    if (validateEmail(email) || (mode === "register" ? validatePassword(password) : !password)) return;
    setBusy(true);
    try {
      if (mode === "login") await signIn(email, password);
      else await signUp(email, password);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={{ gap: spacing.lg }}
      >
        <View style={{ gap: spacing.sm, marginTop: spacing.xl }}>
          <AppText variant="title">Aube</AppText>
          <AppText variant="muted">
            Des rencontres sincères, 100 % gratuites. Pas d&apos;abonnement, pas de limite de likes, pas de
            pub.
          </AppText>
        </View>
        <Row>
          <Chip
            testID="tab-login"
            label="Connexion"
            selected={mode === "login"}
            onPress={() => setMode("login")}
          />
          <Chip
            testID="tab-register"
            label="Créer un compte"
            selected={mode === "register"}
            onPress={() => setMode("register")}
          />
        </Row>
        {error ? <Banner message={error} /> : null}
        <TextField
          testID="email"
          label="Adresse e-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
        />
        <TextField
          testID="password"
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          textContentType={mode === "login" ? "password" : "newPassword"}
          hint={mode === "register" ? "8 caractères minimum." : undefined}
          error={passwordError}
          onSubmitEditing={submit}
        />
        <Button
          testID="submit"
          title={mode === "login" ? "Se connecter" : "Créer mon compte"}
          onPress={submit}
          loading={busy}
        />
        {mode === "login" ? (
          <Button
            title="Mot de passe oublié ?"
            variant="ghost"
            onPress={() => navigation.navigate("ForgotPassword")}
          />
        ) : (
          <AppText variant="muted">
            En créant un compte, vous confirmez avoir 18 ans ou plus. Vos données ne sont jamais vendues.
          </AppText>
        )}
      </KeyboardAvoidingView>
    </Screen>
  );
}
