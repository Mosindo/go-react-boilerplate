import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { ApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { normalizeEmail, parseBirthDate, validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { ScreenContainer } from "../../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text } from "../../shared/ui";
import { spacing } from "../../theme";

type Props = NativeStackScreenProps<AuthStackParamList, "Auth">;
type Mode = "login" | "register";
type FieldErrors = Partial<Record<"email" | "password" | "birth", string>>;

export function AuthScreen({ navigation }: Props) {
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const isRegister = mode === "register";

  const submit = async () => {
    const next: FieldErrors = {};
    const emailError = validateEmail(email);
    if (emailError) next.email = emailError;
    if (isRegister) {
      const passwordError = validatePassword(password);
      if (passwordError) next.password = passwordError;
    } else if (!password) {
      next.password = "Saisissez votre mot de passe.";
    }
    const birth = isRegister ? parseBirthDate(day, month, year) : null;
    if (birth && !birth.ok) next.birth = birth.error;
    setErrors(next);
    setFormError(null);
    if (Object.keys(next).length > 0) return;

    setBusy(true);
    try {
      if (isRegister && birth?.ok) await signUp(normalizeEmail(email), password, birth.iso);
      else await signIn(normalizeEmail(email), password);
    } catch (err) {
      setFormError(err instanceof ApiError || err instanceof Error ? err.message : "Une erreur est survenue.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer scroll contentStyle={styles.content} testID="auth-screen">
      <View style={styles.brand}>
        <Text variant="title" tone="primary" style={styles.logo}>
          Alma
        </Text>
        <Text tone="muted" center>
          Des rencontres sincères. Gratuites, pour de vrai.
        </Text>
      </View>

      <View style={styles.tabs}>
        <Chip testID="auth-mode-login" label="Connexion" selected={!isRegister} onPress={() => setMode("login")} />
        <Chip testID="auth-mode-register" label="Créer un compte" selected={isRegister} onPress={() => setMode("register")} />
      </View>

      {formError ? <Notice testID="auth-error" message={formError} /> : null}

      <FormField label="E-mail" error={errors.email}>
        <Input
          testID="auth-email-input"
          value={email}
          onChangeText={setEmail}
          placeholder="vous@exemple.fr"
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          invalid={!!errors.email}
        />
      </FormField>

      <FormField label="Mot de passe" error={errors.password} hint={isRegister ? "8 caractères minimum." : undefined}>
        <Input
          testID="auth-password-input"
          value={password}
          onChangeText={setPassword}
          placeholder="••••••••"
          secureTextEntry
          autoCapitalize="none"
          autoComplete={isRegister ? "new-password" : "current-password"}
          textContentType={isRegister ? "newPassword" : "password"}
          invalid={!!errors.password}
          onSubmitEditing={isRegister ? undefined : submit}
        />
      </FormField>

      {isRegister ? (
        <FormField
          label="Date de naissance"
          error={errors.birth}
          hint="Alma est réservé aux personnes majeures. Seul votre âge sera visible."
        >
          <View style={styles.birthRow}>
            <Input
              testID="auth-birth-day"
              style={styles.birthSmall}
              value={day}
              onChangeText={(v) => setDay(v.replace(/\D/g, ""))}
              placeholder="JJ"
              keyboardType="number-pad"
              maxLength={2}
              invalid={!!errors.birth}
            />
            <Input
              testID="auth-birth-month"
              style={styles.birthSmall}
              value={month}
              onChangeText={(v) => setMonth(v.replace(/\D/g, ""))}
              placeholder="MM"
              keyboardType="number-pad"
              maxLength={2}
              invalid={!!errors.birth}
            />
            <Input
              testID="auth-birth-year"
              style={styles.birthYear}
              value={year}
              onChangeText={(v) => setYear(v.replace(/\D/g, ""))}
              placeholder="AAAA"
              keyboardType="number-pad"
              maxLength={4}
              invalid={!!errors.birth}
            />
          </View>
        </FormField>
      ) : null}

      <Button testID="auth-submit-button" label={isRegister ? "Créer mon compte" : "Se connecter"} loading={busy} onPress={submit} />

      {!isRegister ? (
        <Pressable
          testID="auth-forgot-link"
          accessibilityRole="link"
          onPress={() => navigation.navigate("ForgotPassword")}
          style={styles.link}
        >
          <Text tone="primary" center style={{ fontWeight: "600" }}>
            Mot de passe oublié ?
          </Text>
        </Pressable>
      ) : (
        <Text variant="caption" tone="muted" center>
          En créant un compte, vous confirmez avoir 18 ans ou plus. Aucun abonnement, aucune publicité obligatoire.
        </Text>
      )}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { justifyContent: "center", paddingVertical: spacing.xl, maxWidth: 440 },
  brand: { alignItems: "center", gap: spacing.xs, marginBottom: spacing.md },
  logo: { fontSize: 44, lineHeight: 52, letterSpacing: 1 },
  tabs: { flexDirection: "row", gap: spacing.sm, justifyContent: "center" },
  birthRow: { flexDirection: "row", gap: spacing.sm },
  birthSmall: { flex: 1, textAlign: "center" },
  birthYear: { flex: 1.6, textAlign: "center" },
  link: { minHeight: 44, justifyContent: "center" }
});
