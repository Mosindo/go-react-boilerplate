import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthProvider";
import { authApi } from "../../api/endpoints";
import { errorMessage, validateEmail, validateNewPassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";
import { Button, FormError, Icon, Screen, Text, TextField, useToast } from "../../ui";

type Props<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen scroll testID="welcome-screen" contentStyle={styles.welcome}>
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Icon name="heart" size={40} color={colors.onPrimary} />
        </View>
        <Text variant="display" center>
          Alba
        </Text>
        <Text center muted>
          Des rencontres sincères. Gratuites, sans abonnement, sans limite de likes, sans publicité.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button
          testID="welcome-register"
          title="Créer un compte"
          onPress={() => navigation.navigate("Register")}
        />
        <Button
          testID="welcome-login"
          title="J'ai déjà un compte"
          variant="secondary"
          onPress={() => navigation.navigate("Login")}
        />
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
    const emailError = validateEmail(email);
    if (emailError || !password) {
      setError(emailError ?? "Entrez votre mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await login(email, password);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="login-screen">
      <Text variant="title">Bon retour</Text>
      <TextField
        testID="login-email"
        label="Adresse e-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
      />
      <TextField
        testID="login-password"
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="current-password"
        textContentType="password"
        onSubmitEditing={submit}
      />
      <FormError testID="login-error" message={error} />
      <Button testID="login-submit" title="Se connecter" onPress={submit} loading={busy} />
      <Button
        testID="login-forgot"
        title="Mot de passe oublié ?"
        variant="ghost"
        onPress={() => navigation.navigate("ForgotPassword", { email })}
      />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { register } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    const problem =
      validateEmail(email) ??
      validateNewPassword(password) ??
      (adult ? null : "Confirmez que vous avez 18 ans ou plus.");
    if (problem) {
      setError(problem);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await register(email, password);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="register-screen">
      <Text variant="title">Créer votre compte</Text>
      <TextField
        testID="register-email"
        label="Adresse e-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
      />
      <TextField
        testID="register-password"
        label="Mot de passe"
        hint="8 caractères minimum."
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="new-password"
        textContentType="newPassword"
      />
      <Pressable
        testID="register-adult"
        accessibilityRole="checkbox"
        accessibilityState={{ checked: adult }}
        onPress={() => setAdult(!adult)}
        style={styles.check}
      >
        <Icon name={adult ? "checkbox" : "square-outline"} />
        <Text style={styles.checkLabel}>J&apos;ai 18 ans ou plus.</Text>
      </Pressable>
      <FormError testID="register-error" message={error} />
      <Button testID="register-submit" title="Continuer" onPress={submit} loading={busy} />
      <Button
        title="J'ai déjà un compte"
        variant="ghost"
        onPress={() => navigation.navigate("Login")}
      />
    </Screen>
  );
}

export function ForgotPasswordScreen({ route, navigation }: Props<"ForgotPassword">) {
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendCode = async () => {
    const problem = validateEmail(email);
    if (problem) {
      setError(problem);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setSent(true);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const problem =
      code.trim().length < 6 ? "Entrez le code reçu par e-mail." : validateNewPassword(password);
    if (problem) {
      setError(problem);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      toast.show({ message: "Mot de passe modifié. Connectez-vous.", kind: "success" });
      navigation.navigate("Login");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Text variant="title">Récupérer mon compte</Text>
      <TextField
        testID="forgot-email"
        label="Adresse e-mail"
        value={email}
        onChangeText={setEmail}
        editable={!sent}
        autoCapitalize="none"
        keyboardType="email-address"
      />
      {sent ? (
        <>
          <Text muted>
            Si un compte existe pour cette adresse, un code à 8 caractères vient d&apos;être envoyé.
            Il expire dans 30 minutes.
          </Text>
          <TextField
            testID="forgot-code"
            label="Code reçu"
            value={code}
            onChangeText={(v) => setCode(v.toUpperCase())}
            autoCapitalize="characters"
            autoCorrect={false}
          />
          <TextField
            testID="forgot-password"
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoComplete="new-password"
          />
        </>
      ) : null}
      <FormError testID="forgot-error" message={error} />
      {sent ? (
        <Button
          testID="forgot-reset"
          title="Changer mon mot de passe"
          onPress={reset}
          loading={busy}
        />
      ) : (
        <Button testID="forgot-send" title="Envoyer le code" onPress={sendCode} loading={busy} />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  welcome: { justifyContent: "space-between", minHeight: 520 },
  hero: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.lg,
    paddingVertical: spacing.xxl,
  },
  logo: { width: 88, height: 88, borderRadius: 28, alignItems: "center", justifyContent: "center" },
  actions: { gap: spacing.md },
  check: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 44 },
  checkLabel: { flex: 1 },
});
