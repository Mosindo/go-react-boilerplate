import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useRef, useState } from "react";
import { Pressable, View, type TextInput } from "react-native";
import { useAuth } from "../hooks/useAuth";
import { errorMessage } from "../lib/errors";
import { isValidEmail, normalizeEmail, passwordProblem } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { Button, Input, Notice, Text } from "../shared/ui";
import { ScreenContainer } from "../shared/layout";
import { radii, spacing, useStyles, type Colors } from "../theme";

type Mode = "login" | "register";

/** Sign in / create an account on one screen. Everything is validated again by the server. */
export default function AuthScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { login, register } = useAuth();
  const styles = useStyles(makeStyles);
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const emailError = touched && !isValidEmail(email) ? "Entrez une adresse email valide." : null;
  const passwordError = touched ? (mode === "register" ? passwordProblem(password) : password ? null : "Entrez votre mot de passe.") : null;

  const submit = async () => {
    setTouched(true);
    setFormError(null);
    if (!isValidEmail(email) || (mode === "register" ? passwordProblem(password) : !password)) return;
    setSubmitting(true);
    try {
      if (mode === "login") await login(normalizeEmail(email), password);
      else await register(normalizeEmail(email), password);
    } catch (error) {
      setFormError(errorMessage(error));
      setSubmitting(false);
    }
  };

  return (
    <ScreenContainer scroll testID="auth-screen">
      <View style={styles.hero}>
        <Text variant="hero" tone="primary">
          Lueur
        </Text>
        <Text variant="heading" style={styles.tagline}>
          Des rencontres sincères.{"\n"}Gratuites, pour de vrai.
        </Text>
        <Text tone="muted">Pas d'abonnement, pas de boost payant, pas de limite de likes.</Text>
      </View>

      <View style={styles.segment} accessibilityRole="tablist">
        {(["login", "register"] as const).map((value) => (
          <Pressable
            key={value}
            testID={`auth-mode-${value}`}
            accessibilityRole="tab"
            accessibilityState={{ selected: mode === value }}
            onPress={() => {
              setMode(value);
              setFormError(null);
            }}
            style={[styles.segmentItem, mode === value && styles.segmentActive]}
          >
            <Text variant="bodyStrong" tone={mode === value ? "default" : "muted"}>
              {value === "login" ? "Connexion" : "Inscription"}
            </Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.form}>
        <Input
          testID="auth-email"
          label="Email"
          value={email}
          onChangeText={setEmail}
          error={emailError}
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
          keyboardType="email-address"
          textContentType="emailAddress"
          returnKeyType="next"
          onSubmitEditing={() => passwordRef.current?.focus()}
        />
        <Input
          ref={passwordRef}
          testID="auth-password"
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          error={passwordError}
          hint={mode === "register" ? "8 caractères minimum." : undefined}
          secureTextEntry
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          textContentType={mode === "login" ? "password" : "newPassword"}
          returnKeyType="go"
          onSubmitEditing={submit}
        />
        {formError ? <Notice kind="error" message={formError} /> : null}
        <Button testID="auth-submit" label={mode === "login" ? "Se connecter" : "Créer mon compte"} onPress={submit} loading={submitting} />
        {mode === "login" ? (
          <Button label="Mot de passe oublié ?" variant="ghost" compact onPress={() => navigation.navigate("ForgotPassword")} testID="auth-forgot" />
        ) : (
          <Text variant="caption" tone="muted" center>
            En créant un compte, vous confirmez avoir au moins 18 ans.
          </Text>
        )}
      </View>
    </ScreenContainer>
  );
}

const makeStyles = (colors: Colors) => ({
  hero: { gap: spacing.sm, paddingTop: spacing.xl, paddingBottom: spacing.md },
  tagline: { marginTop: spacing.sm },
  segment: { flexDirection: "row", backgroundColor: colors.surfaceMuted, borderRadius: radii.pill, padding: 4 } as const,
  segmentItem: { flex: 1, minHeight: 44, alignItems: "center", justifyContent: "center", borderRadius: radii.pill } as const,
  segmentActive: { backgroundColor: colors.surface },
  form: { gap: spacing.lg }
});
