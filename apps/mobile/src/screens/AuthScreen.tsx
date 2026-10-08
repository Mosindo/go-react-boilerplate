import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { login, register } from "../api/auth";
import { describeError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import type { AuthStackParamList } from "../navigation/types";
import { ScreenContainer, SafeAreaLayout } from "../shared/layout";
import { Button, FormField, Notice, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";
import { validateEmail, validatePassword } from "../utils/validation";

type Mode = "login" | "register";

export default function AuthScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Auth">) {
  const { colors } = useTheme();
  const { signIn, sessionExpired } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [touched, setTouched] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const emailError = touched ? validateEmail(email) : null;
  const passwordError = touched && mode === "register" ? validatePassword(password) : touched && !password ? "Saisissez votre mot de passe." : null;

  const submit = async () => {
    setTouched(true);
    setError(null);
    if (validateEmail(email) || (mode === "register" ? validatePassword(password) : !password)) return;
    setSubmitting(true);
    try {
      const session = mode === "login" ? await login(email.trim(), password) : await register(email.trim(), password);
      await signIn(session);
    } catch (e) {
      setError(describeError(e));
      setSubmitting(false);
    }
  };

  return (
    <SafeAreaLayout>
      <ScreenContainer>
        <View style={styles.brand}>
          <View style={[styles.logo, { backgroundColor: colors.primary }]}>
            <Ionicons color={colors.onPrimary} name="heart" size={28} />
          </View>
          <Text accessibilityRole="header" variant="title">
            Lien
          </Text>
          <Text align="center" tone="muted">
            Des rencontres sincères. Gratuit, pour toujours : pas d'abonnement, pas de limite.
          </Text>
        </View>

        <View accessibilityRole="tablist" style={[styles.segment, { backgroundColor: colors.surfaceMuted }]}>
          {(["login", "register"] as const).map((value) => (
            <Pressable
              accessibilityRole="tab"
              accessibilityState={{ selected: mode === value }}
              key={value}
              onPress={() => {
                setMode(value);
                setError(null);
              }}
              style={[styles.segmentItem, mode === value ? { backgroundColor: colors.surface } : null]}
            >
              <Text weight="semibold" tone={mode === value ? "default" : "muted"}>
                {value === "login" ? "Connexion" : "Créer un compte"}
              </Text>
            </Pressable>
          ))}
        </View>

        {sessionExpired && !error ? <Notice message="Votre session a expiré. Reconnectez-vous." /> : null}
        {error ? <Notice message={error} tone="error" /> : null}

        <FormField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          keyboardType="email-address"
          label="Adresse e-mail"
          onChangeText={setEmail}
          placeholder="vous@exemple.fr"
          textContentType="emailAddress"
          value={email}
        />
        <View>
          <FormField
            autoCapitalize="none"
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            error={passwordError}
            hint={mode === "register" ? "8 caractères minimum, avec des lettres et des chiffres ou symboles." : undefined}
            label="Mot de passe"
            onChangeText={setPassword}
            onSubmitEditing={submit}
            secureTextEntry={!showPassword}
            textContentType={mode === "login" ? "password" : "newPassword"}
            value={password}
          />
          <Pressable
            accessibilityLabel={showPassword ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            accessibilityRole="button"
            hitSlop={8}
            onPress={() => setShowPassword((v) => !v)}
            style={styles.eye}
          >
            <Ionicons color={colors.textMuted} name={showPassword ? "eye-off-outline" : "eye-outline"} size={22} />
          </Pressable>
        </View>

        {mode === "register" ? (
          <Text tone="subtle" variant="caption">
            En créant un compte, vous confirmez avoir 18 ans ou plus.
          </Text>
        ) : null}

        <Button fullWidth loading={submitting} onPress={submit} size="lg">
          {mode === "login" ? "Se connecter" : "Créer mon compte"}
        </Button>
        {mode === "login" ? (
          <Button onPress={() => navigation.navigate("Forgot", { email: email.trim() || undefined })} variant="ghost">
            Mot de passe oublié ?
          </Button>
        ) : null}
      </ScreenContainer>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  brand: { alignItems: "center", gap: spacing.sm, paddingTop: spacing.xl, paddingBottom: spacing.md },
  logo: { width: 60, height: 60, borderRadius: 30, alignItems: "center", justifyContent: "center" },
  segment: { flexDirection: "row", borderRadius: radii.pill, padding: 4 },
  segmentItem: { flex: 1, alignItems: "center", justifyContent: "center", minHeight: 44, borderRadius: radii.pill },
  eye: { position: "absolute", right: spacing.md, top: 31 }
});
