import React, { useState } from "react";
import { View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { ApiError } from "../api/client";
import { authApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthProvider";
import { errorMessage } from "../components/forms";
import { Button, ErrorText, Screen, Text, TextField } from "../components/ui";
import { normalizeEmail, validateEmail, validatePassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme/theme";

type Props<K extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, K>;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  const t = useTheme();
  return (
    <Screen scroll>
      <View style={{ flex: 1, justifyContent: "center", paddingVertical: spacing.xxl }}>
        <View style={{ alignItems: "center", marginBottom: spacing.xxl }}>
          <View
            style={{ width: 88, height: 88, borderRadius: 44, backgroundColor: t.primary, alignItems: "center", justifyContent: "center", marginBottom: spacing.xl }}
          >
            <Ionicons name="sunny" size={46} color={t.onPrimary} />
          </View>
          <Text variant="display" style={{ textAlign: "center" }}>
            Aurore
          </Text>
          <Text muted style={{ textAlign: "center", marginTop: spacing.md, maxWidth: 320 }}>
            Des rencontres sincères, sans abonnement, sans publicité, sans limite artificielle. Tout est gratuit.
          </Text>
        </View>
        <Button label="Créer mon compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} style={{ marginTop: spacing.md }} testID="welcome-login" />
        <Text variant="caption" muted style={{ textAlign: "center", marginTop: spacing.xl }}>
          Aurore est réservée aux personnes majeures.
        </Text>
      </View>
    </Screen>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const next = { email: validateEmail(email) ?? undefined, password: password ? undefined : "Entrez votre mot de passe." };
    setErrors(next);
    if (next.email || next.password) return;
    setBusy(true);
    try {
      await signIn(normalizeEmail(email), password);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={{ marginTop: spacing.xl, marginBottom: spacing.xl }}>
        Bon retour
      </Text>
      <TextField label="Email" value={email} onChangeText={setEmail} error={errors.email} autoCapitalize="none" autoComplete="email" keyboardType="email-address" testID="login-email" />
      <TextField label="Mot de passe" value={password} onChangeText={setPassword} error={errors.password} secureTextEntry autoComplete="current-password" onSubmitEditing={() => void submit()} testID="login-password" />
      {errors.form ? <ErrorText style={{ marginBottom: spacing.md }}>{errors.form}</ErrorText> : null}
      <Button label="Se connecter" onPress={() => void submit()} loading={busy} testID="login-submit" />
      <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("Forgot", { email })} style={{ marginTop: spacing.sm }} testID="login-forgot" />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { signUp } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string; adult?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      adult: adult ? undefined : "Vous devez confirmer avoir 18 ans ou plus."
    };
    setErrors(next);
    if (next.email || next.password || next.adult) return;
    setBusy(true);
    try {
      await signUp(normalizeEmail(email), password);
    } catch (e) {
      setErrors({ form: e instanceof ApiError && e.status === 409 ? "Un compte existe déjà avec cet email." : errorMessage(e) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={{ marginTop: spacing.xl, marginBottom: spacing.sm }}>
        Créer mon compte
      </Text>
      <Text muted style={{ marginBottom: spacing.xl }}>
        Gratuit, pour toujours. Vous créerez votre profil juste après.
      </Text>
      <TextField label="Email" value={email} onChangeText={setEmail} error={errors.email} autoCapitalize="none" autoComplete="email" keyboardType="email-address" testID="register-email" />
      <TextField label="Mot de passe" value={password} onChangeText={setPassword} error={errors.password} hint="8 caractères minimum" secureTextEntry autoComplete="new-password" testID="register-password" />
      <Button
        label={adult ? "✓ Je confirme avoir 18 ans ou plus" : "Je confirme avoir 18 ans ou plus"}
        variant={adult ? "primary" : "secondary"}
        onPress={() => setAdult((a) => !a)}
        testID="register-adult"
      />
      {errors.adult ? <ErrorText style={{ marginTop: spacing.sm }}>{errors.adult}</ErrorText> : null}
      {errors.form ? <ErrorText style={{ marginTop: spacing.md }}>{errors.form}</ErrorText> : null}
      <Button label="Continuer" onPress={() => void submit()} loading={busy} style={{ marginTop: spacing.xl }} testID="register-submit" />
      <Button label="J'ai déjà un compte" variant="ghost" onPress={() => navigation.navigate("Login")} style={{ marginTop: spacing.sm }} />
    </Screen>
  );
}

export function ForgotScreen({ navigation, route }: Props<"Forgot">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; code?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const request = async () => {
    const emailError = validateEmail(email);
    setErrors({ email: emailError ?? undefined });
    if (emailError) return;
    setBusy(true);
    try {
      await authApi.forgot(normalizeEmail(email));
      setSent(true);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const next = {
      code: code.trim().length >= 6 ? undefined : "Entrez le code reçu par email.",
      password: validatePassword(password) ?? undefined
    };
    setErrors(next);
    if (next.code || next.password) return;
    setBusy(true);
    try {
      await authApi.reset(normalizeEmail(email), code.trim(), password);
      navigation.navigate("Login");
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Text variant="title" style={{ marginTop: spacing.xl, marginBottom: spacing.sm }}>
        Récupérer mon compte
      </Text>
      {!sent ? (
        <>
          <Text muted style={{ marginBottom: spacing.xl }}>
            Entrez votre email : si un compte existe, nous vous envoyons un code valable 30 minutes.
          </Text>
          <TextField label="Email" value={email} onChangeText={setEmail} error={errors.email} autoCapitalize="none" keyboardType="email-address" testID="forgot-email" />
          {errors.form ? <ErrorText style={{ marginBottom: spacing.md }}>{errors.form}</ErrorText> : null}
          <Button label="Envoyer le code" onPress={() => void request()} loading={busy} testID="forgot-submit" />
        </>
      ) : (
        <>
          <Text muted style={{ marginBottom: spacing.xl }}>
            Si un compte existe pour {normalizeEmail(email)}, un code vient d&apos;être envoyé. Entrez-le avec votre nouveau mot de passe.
          </Text>
          <TextField label="Code de récupération" value={code} onChangeText={(v) => setCode(v.toUpperCase())} error={errors.code} autoCapitalize="characters" maxLength={12} testID="reset-code" />
          <TextField label="Nouveau mot de passe" value={password} onChangeText={setPassword} error={errors.password} secureTextEntry hint="8 caractères minimum" testID="reset-password" />
          {errors.form ? <ErrorText style={{ marginBottom: spacing.md }}>{errors.form}</ErrorText> : null}
          <Button label="Changer mon mot de passe" onPress={() => void reset()} loading={busy} testID="reset-submit" />
          <Button label="Renvoyer un code" variant="ghost" onPress={() => void request()} style={{ marginTop: spacing.sm }} />
        </>
      )}
    </Screen>
  );
}
