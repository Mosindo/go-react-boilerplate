import React, { useRef, useState } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { ApiError } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../auth/AuthContext";
import { parseBirthDate } from "../../lib/birthDate";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import type { RootParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";

type Nav = NativeStackNavigationProp<RootParamList>;

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : "Une erreur est survenue. Réessaie.";
}

export function WelcomeScreen() {
  const nav = useNavigation<Nav>();
  const { colors } = useTheme();
  return (
    <Screen padded={false} edges={["bottom"]}>
      <LinearGradient colors={[colors.primarySoft, colors.background]} style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Text variant="display" tone="onPrimary">
            E
          </Text>
        </View>
        <Text variant="display" align="center" style={styles.brand}>
          Ember
        </Text>
        <Text tone="textMuted" align="center" style={styles.tagline}>
          Des rencontres sincères, sans abonnement, sans paywall, sans pub. Gratuit, pour de vrai.
        </Text>
      </LinearGradient>
      <View style={styles.actions}>
        <Button title="Créer un compte" onPress={() => nav.navigate("Register")} testID="welcome-register" />
        <Button
          title="J'ai déjà un compte"
          variant="secondary"
          onPress={() => nav.navigate("Login")}
          testID="welcome-login"
        />
        <Text variant="caption" tone="textSubtle" align="center">
          Réservé aux personnes majeures.
        </Text>
      </View>
    </Screen>
  );
}

export function LoginScreen() {
  const nav = useNavigation<Nav>();
  const { signIn } = useAuth();
  const passwordRef = useRef<TextInput>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const emailError = validateEmail(email);
    if (emailError || !password) {
      setError(emailError ?? "Entre ton mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(await authApi.login(normalizeEmail(email), password));
    } catch (err) {
      setError(
        messageOf(err) === "invalid credentials" ? "E-mail ou mot de passe incorrect." : messageOf(err)
      );
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={styles.title}>
        Content de te revoir
      </Text>
      <TextField
        label="Adresse e-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        returnKeyType="next"
        onSubmitEditing={() => passwordRef.current?.focus()}
        testID="login-email"
      />
      <TextField
        ref={passwordRef}
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="current-password"
        returnKeyType="go"
        onSubmitEditing={submit}
        error={error}
        testID="login-password"
      />
      <Button title="Se connecter" onPress={submit} loading={busy} testID="login-submit" />
      <Button title="Mot de passe oublié ?" variant="ghost" onPress={() => nav.navigate("ForgotPassword")} />
    </Screen>
  );
}

export function RegisterScreen() {
  const { signIn } = useAuth();
  const [form, setForm] = useState({ firstName: "", email: "", password: "", day: "", month: "", year: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof typeof form | "form", string>>>({});
  const [busy, setBusy] = useState(false);
  const set = (key: keyof typeof form) => (value: string) => setForm((f) => ({ ...f, [key]: value }));

  const submit = async () => {
    const next: typeof errors = {};
    const firstName = form.firstName.trim();
    if (!firstName || firstName.length > 40) next.firstName = "Entre ton prénom (40 caractères max).";
    const emailError = validateEmail(form.email);
    if (emailError) next.email = emailError;
    const passwordError = validatePassword(form.password);
    if (passwordError) next.password = passwordError;
    const birth = parseBirthDate(form.day, form.month, form.year);
    if (birth.error) next.day = birth.error;
    setErrors(next);
    if (Object.keys(next).length > 0 || !birth.iso) return;

    setBusy(true);
    try {
      await signIn(
        await authApi.register({
          email: normalizeEmail(form.email),
          password: form.password,
          firstName,
          birthDate: birth.iso
        })
      );
    } catch (err) {
      const message = messageOf(err);
      setErrors({
        form: message === "email already exists" ? "Un compte existe déjà avec cet e-mail." : message
      });
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={styles.title}>
        Crée ton compte
      </Text>
      <TextField
        label="Prénom"
        value={form.firstName}
        onChangeText={set("firstName")}
        autoComplete="given-name"
        error={errors.firstName}
        testID="register-firstname"
      />
      <TextField
        label="Adresse e-mail"
        value={form.email}
        onChangeText={set("email")}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        error={errors.email}
        testID="register-email"
      />
      <TextField
        label="Mot de passe"
        value={form.password}
        onChangeText={set("password")}
        secureTextEntry
        autoComplete="new-password"
        helper="8 caractères minimum."
        error={errors.password}
        testID="register-password"
      />
      <Text variant="label" tone="textMuted" style={styles.dobLabel}>
        Date de naissance
      </Text>
      <View style={styles.dob}>
        <View style={styles.dobSmall}>
          <TextField
            label="Jour"
            value={form.day}
            onChangeText={set("day")}
            keyboardType="number-pad"
            maxLength={2}
            placeholder="JJ"
            testID="register-day"
          />
        </View>
        <View style={styles.dobSmall}>
          <TextField
            label="Mois"
            value={form.month}
            onChangeText={set("month")}
            keyboardType="number-pad"
            maxLength={2}
            placeholder="MM"
            testID="register-month"
          />
        </View>
        <View style={styles.dobYear}>
          <TextField
            label="Année"
            value={form.year}
            onChangeText={set("year")}
            keyboardType="number-pad"
            maxLength={4}
            placeholder="AAAA"
            testID="register-year"
          />
        </View>
      </View>
      {errors.day ? (
        <Text variant="caption" tone="danger" style={styles.formError}>
          {errors.day}
        </Text>
      ) : null}
      {errors.form ? (
        <Text tone="danger" style={styles.formError} accessibilityLiveRegion="polite">
          {errors.form}
        </Text>
      ) : null}
      <Text variant="caption" tone="textSubtle" style={styles.formError}>
        Ton âge est affiché sur ton profil, pas ta date de naissance. Ember est et restera gratuit.
      </Text>
      <Button title="Créer mon compte" onPress={submit} loading={busy} testID="register-submit" />
    </Screen>
  );
}

export function ForgotPasswordScreen() {
  const nav = useNavigation<Nav>();
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [step, setStep] = useState<"email" | "reset" | "done">("email");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const sendCode = async () => {
    const emailError = validateEmail(email);
    if (emailError) return setError(emailError);
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(normalizeEmail(email));
      setStep("reset");
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const passwordError = validatePassword(password);
    if (code.trim().length !== 10) return setError("Le code contient 10 caractères.");
    if (passwordError) return setError(passwordError);
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(code.trim(), password);
      setStep("done");
    } catch (err) {
      setError(messageOf(err) === "invalid or expired code" ? "Code invalide ou expiré." : messageOf(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={styles.title}>
        Récupérer mon compte
      </Text>
      {step === "email" ? (
        <>
          <Text tone="textMuted" style={styles.formError}>
            Entre ton e-mail : nous t&apos;enverrons un code valable 1 heure.
          </Text>
          <TextField
            label="Adresse e-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
            error={error}
            testID="forgot-email"
          />
          <Button title="Envoyer le code" onPress={sendCode} loading={busy} testID="forgot-send" />
        </>
      ) : null}
      {step === "reset" ? (
        <>
          <Text tone="textMuted" style={styles.formError}>
            Si un compte existe pour {normalizeEmail(email)}, un code vient de lui être envoyé.
          </Text>
          <TextField
            label="Code reçu"
            value={code}
            onChangeText={setCode}
            autoCapitalize="characters"
            maxLength={10}
            testID="forgot-code"
          />
          <TextField
            label="Nouveau mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            error={error}
            testID="forgot-password"
          />
          <Button title="Changer mon mot de passe" onPress={reset} loading={busy} testID="forgot-reset" />
        </>
      ) : null}
      {step === "done" ? (
        <>
          <Text style={styles.formError}>Mot de passe mis à jour. Tu peux maintenant te connecter.</Text>
          <Button title="Se connecter" onPress={() => nav.navigate("Login")} />
        </>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", paddingHorizontal: spacing.lg },
  logo: { width: 84, height: 84, borderRadius: 28, alignItems: "center", justifyContent: "center" },
  brand: { marginTop: spacing.md },
  tagline: { marginTop: spacing.sm, maxWidth: 320 },
  actions: { padding: spacing.lg, gap: spacing.sm },
  title: { marginVertical: spacing.md },
  dobLabel: { marginBottom: 0 },
  dob: { flexDirection: "row", gap: spacing.sm },
  dobSmall: { flex: 1 },
  dobYear: { flex: 1.6 },
  formError: { marginBottom: spacing.md }
});
