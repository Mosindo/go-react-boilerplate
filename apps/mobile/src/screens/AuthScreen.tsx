import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useSession } from "../state/session";
import { errorMessage } from "../hooks/queries";
import { validateEmail, validateNewPassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { Button, Chip, Input, Screen, Text, spacing, useTheme } from "../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Auth">;

export default function AuthScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { login, register } = useSession();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password:
        mode === "register"
          ? (validateNewPassword(password) ?? undefined)
          : password
            ? undefined
            : "Saisissez votre mot de passe."
    };
    if (next.email || next.password) {
      setErrors(next);
      return;
    }
    setErrors({});
    setBusy(true);
    try {
      if (mode === "login") {
        await login(email.trim(), password);
      } else {
        await register(email.trim(), password);
      }
    } catch (error) {
      setErrors({ form: errorMessage(error) });
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="auth-screen" topInset>
      <View style={styles.brand}>
        <Text variant="display" tone="primary">
          Alba
        </Text>
        <Text tone="muted">Des rencontres sincères, 100 % gratuites. Sans abonnement, sans limite.</Text>
      </View>

      <View style={styles.modes}>
        <Chip label="Connexion" onPress={() => setMode("login")} selected={mode === "login"} testID="mode-login" />
        <Chip
          label="Créer un compte"
          onPress={() => setMode("register")}
          selected={mode === "register"}
          testID="mode-register"
        />
      </View>

      <View style={styles.form}>
        <Input
          autoCapitalize="none"
          autoComplete="email"
          error={errors.email}
          keyboardType="email-address"
          label="Adresse e-mail"
          onChangeText={setEmail}
          testID="auth-email"
          textContentType="emailAddress"
          value={email}
        />
        <Input
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={errors.password}
          hint={mode === "register" ? "10 caractères minimum." : undefined}
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureTextEntry
          testID="auth-password"
          textContentType={mode === "login" ? "password" : "newPassword"}
          value={password}
        />
        {errors.form ? (
          <Text
            accessibilityRole="alert"
            style={[styles.formError, { backgroundColor: colors.dangerSoft }]}
            tone="danger"
          >
            {errors.form}
          </Text>
        ) : null}
        <Button
          label={mode === "login" ? "Se connecter" : "Créer mon compte"}
          loading={busy}
          onPress={submit}
          testID="auth-submit"
        />
        {mode === "login" ? (
          <Button
            label="Mot de passe oublié ?"
            onPress={() => navigation.navigate("Forgot")}
            testID="auth-forgot"
            variant="ghost"
          />
        ) : null}
      </View>

      {mode === "register" ? (
        <Text tone="muted" variant="caption">
          En créant un compte, vous confirmez avoir 18 ans ou plus et vous engagez à respecter les autres membres.
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  brand: { gap: spacing.sm, marginTop: spacing.xxl, marginBottom: spacing.xl },
  modes: { flexDirection: "row", gap: spacing.sm, marginBottom: spacing.lg },
  form: { gap: spacing.lg, marginBottom: spacing.lg },
  formError: { padding: spacing.md, borderRadius: 12, overflow: "hidden" }
});
