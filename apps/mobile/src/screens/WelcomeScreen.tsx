import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Button } from "../components/Button";
import { Screen } from "../components/Screen";
import { Text } from "../components/Text";
import { TextField } from "../components/TextField";
import { useAuth } from "../auth/AuthContext";
import { validEmail, validPassword } from "../lib/format";
import type { AuthStackParamList } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";

type Mode = "login" | "register";

export function WelcomeScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AuthStackParamList>>();
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    const next: typeof errors = {};
    if (!validEmail(email)) next.email = "Entrez une adresse email valide.";
    if (mode === "register") {
      const issue = validPassword(password);
      if (issue) next.password = issue;
    } else if (!password) {
      next.password = "Entrez votre mot de passe.";
    }
    setErrors(next);
    if (Object.keys(next).length > 0) return;

    setLoading(true);
    try {
      if (mode === "login") await signIn(email, password);
      else await signUp(email, password);
    } catch (err) {
      setErrors({ form: err instanceof Error ? err.message : "Connexion impossible." });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll contentStyle={styles.content}>
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: theme.primary }]}>
          <Text variant="title" color={theme.onPrimary} accessibilityElementsHidden>
            L
          </Text>
        </View>
        <Text variant="display">Lien</Text>
        <Text muted>Des rencontres sincères. Gratuites, pour toujours.</Text>
      </View>

      <View style={[styles.tabs, { backgroundColor: theme.surfaceAlt }]} accessibilityRole="tablist">
        {(["login", "register"] as const).map((m) => (
          <Pressable
            key={m}
            accessibilityRole="tab"
            accessibilityState={{ selected: mode === m }}
            onPress={() => {
              setMode(m);
              setErrors({});
            }}
            style={[styles.tab, mode === m ? { backgroundColor: theme.surface } : null]}
            testID={`tab-${m}`}
          >
            <Text variant="bodyStrong" color={mode === m ? theme.text : theme.textMuted}>
              {m === "login" ? "Connexion" : "Inscription"}
            </Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.form}>
        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          error={errors.email}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          error={errors.password}
          hint={mode === "register" ? "8 caractères minimum, avec une lettre et un chiffre." : undefined}
          secure
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          textContentType={mode === "login" ? "password" : "newPassword"}
          onSubmitEditing={submit}
          testID="password"
        />
        {errors.form ? (
          <Text color={theme.danger} accessibilityRole="alert" testID="form-error">
            {errors.form}
          </Text>
        ) : null}
        <Button
          label={mode === "login" ? "Se connecter" : "Créer mon compte"}
          onPress={submit}
          loading={loading}
          testID="submit-auth"
        />
        {mode === "login" ? (
          <Button
            label="Mot de passe oublié ?"
            variant="ghost"
            small
            onPress={() => navigation.navigate("Forgot", { email })}
          />
        ) : (
          <Text variant="caption" muted center>
            En créant un compte, vous confirmez avoir 18 ans ou plus.
          </Text>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { justifyContent: "center", maxWidth: 480, width: "100%", alignSelf: "center", gap: spacing.xl },
  hero: { alignItems: "center", gap: spacing.sm },
  logo: {
    width: 64,
    height: 64,
    borderRadius: 22,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  tabs: { flexDirection: "row", borderRadius: radius.pill, padding: 4 },
  tab: { flex: 1, alignItems: "center", paddingVertical: spacing.sm + 2, borderRadius: radius.pill },
  form: { gap: spacing.lg }
});
