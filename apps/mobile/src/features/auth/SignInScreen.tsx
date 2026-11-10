import React, { useRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View } from "react-native";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordIssue } from "../../utils/validation";

export function SignInScreen({ navigation }: AuthScreenProps<"SignIn">) {
  const theme = useTheme();
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<"signUp" | "signIn">("signUp");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accepted, setAccepted] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const emailError = submitted && !isValidEmail(email) ? "Adresse email invalide." : null;
  const pwdError = submitted ? (mode === "signUp" ? passwordIssue(password) : password ? null : "Mot de passe requis.") : null;

  const submit = async () => {
    setSubmitted(true);
    setError(null);
    if (!isValidEmail(email) || (mode === "signUp" ? passwordIssue(password) : !password)) return;
    if (mode === "signUp" && !accepted) {
      setError("Confirmez que vous avez 18 ans ou plus et acceptez les règles de la communauté.");
      return;
    }
    setLoading(true);
    try {
      await (mode === "signUp" ? signUp(email, password) : signIn(email, password));
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll edges={["top", "bottom", "left", "right"]}>
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: theme.colors.primary }]}>
          <View style={[styles.logoCore, { backgroundColor: theme.colors.onPrimary }]} />
        </View>
        <Text variant="display">Lueur</Text>
        <Text tone="muted">Des rencontres sincères, entièrement gratuites. Sans abonnement, sans limite de likes.</Text>
      </View>

      <View accessibilityRole="tablist" style={[styles.switcher, { backgroundColor: theme.colors.surfaceAlt }]}>
        {(["signUp", "signIn"] as const).map((m) => (
          <Pressable
            accessibilityRole="tab"
            accessibilityState={{ selected: mode === m }}
            key={m}
            onPress={() => {
              setMode(m);
              setError(null);
              setSubmitted(false);
            }}
            style={[styles.tab, mode === m && { backgroundColor: theme.colors.surface }, mode === m && theme.shadow(1)]}
          >
            <Text tone={mode === m ? "default" : "muted"} variant="label">
              {m === "signUp" ? "Créer un compte" : "Se connecter"}
            </Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          inputMode="email"
          keyboardType="email-address"
          label="Email"
          onChangeText={setEmail}
          onSubmitEditing={() => passwordRef.current?.focus()}
          placeholder="vous@exemple.fr"
          returnKeyType="next"
          testID="auth-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          ref={passwordRef}
          autoComplete={mode === "signUp" ? "new-password" : "current-password"}
          error={pwdError}
          hint={mode === "signUp" ? "8 caractères minimum, avec une lettre et un chiffre." : undefined}
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          returnKeyType="go"
          secureTextEntry
          testID="auth-password"
          textContentType={mode === "signUp" ? "newPassword" : "password"}
          value={password}
        />

        {mode === "signUp" ? (
          <Pressable
            accessibilityRole="checkbox"
            accessibilityState={{ checked: accepted }}
            onPress={() => setAccepted((v) => !v)}
            style={styles.consent}
            testID="auth-consent"
          >
            <View
              style={[
                styles.box,
                { borderColor: theme.colors.borderStrong },
                accepted && { backgroundColor: theme.colors.primary, borderColor: theme.colors.primary }
              ]}
            >
              {accepted ? (
                <Text tone="onPrimary" variant="caption">
                  ✓
                </Text>
              ) : null}
            </View>
            <Text style={styles.flex} tone="muted" variant="small">
              J’ai 18 ans ou plus et je m’engage à respecter les autres membres : bienveillance, consentement, aucune forme de harcèlement.
            </Text>
          </Pressable>
        ) : (
          <Pressable accessibilityRole="link" onPress={() => navigation.navigate("ForgotPassword", { email })} style={styles.forgot}>
            <Text tone="primary" variant="label">
              Mot de passe oublié ?
            </Text>
          </Pressable>
        )}

        {error ? (
          <Text accessibilityLiveRegion="polite" tone="danger" variant="small">
            {error}
          </Text>
        ) : null}

        <Button label={mode === "signUp" ? "Créer mon compte" : "Se connecter"} loading={loading} onPress={submit} testID="auth-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { gap: 10, paddingTop: 40, paddingBottom: 28 },
  logo: { width: 52, height: 52, borderRadius: 26, alignItems: "center", justifyContent: "center", marginBottom: 8 },
  logoCore: { width: 18, height: 18, borderRadius: 9, opacity: 0.9 },
  switcher: { flexDirection: "row", borderRadius: 999, padding: 4, marginBottom: 24 },
  tab: { flex: 1, alignItems: "center", paddingVertical: 10, borderRadius: 999 },
  form: { gap: 14 },
  consent: { flexDirection: "row", gap: 12, alignItems: "flex-start" },
  box: { width: 22, height: 22, borderRadius: 6, borderWidth: 1.5, alignItems: "center", justifyContent: "center", marginTop: 2 },
  flex: { flex: 1 },
  forgot: { alignSelf: "flex-end", paddingVertical: 4 }
});
