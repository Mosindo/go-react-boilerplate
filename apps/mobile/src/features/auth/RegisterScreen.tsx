import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { authApi } from "../../api/auth";
import type { RootScreenProps } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { describeError } from "../../utils/errors";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function RegisterScreen({ navigation }: RootScreenProps<"Register">) {
  const { colors, spacing } = useTheme();
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accepted, setAccepted] = useState(false);
  const [touched, setTouched] = useState(false);

  const emailError = touched && !EMAIL_PATTERN.test(email.trim()) ? "Saisissez une adresse e-mail valide." : null;
  const passwordError =
    touched && (password.length < 8 || password.length > 72) ? "8 caractères minimum (72 maximum)." : null;

  const register = useMutation({
    mutationFn: () => authApi.register(email.trim(), password),
    onSuccess: signIn
  });

  const submit = () => {
    setTouched(true);
    if (!EMAIL_PATTERN.test(email.trim()) || password.length < 8 || password.length > 72 || !accepted) {
      return;
    }
    register.mutate();
  };

  return (
    <Screen scroll testID="register-screen">
      <Header back />
      <View style={{ gap: spacing.lg, marginTop: spacing.md }}>
        <View style={{ gap: spacing.xs }}>
          <Text variant="title" accessibilityRole="header">
            Créer votre compte
          </Text>
          <Text tone="muted">Gratuit, pour toujours. Votre e-mail ne sera jamais affiché.</Text>
        </View>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
          testID="register-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="new-password"
          textContentType="newPassword"
          error={passwordError}
          hint="8 caractères minimum."
          testID="register-password"
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: accepted }}
          onPress={() => setAccepted((v) => !v)}
          style={styles.consent}
          testID="register-consent"
        >
          <Ionicons
            name={accepted ? "checkbox" : "square-outline"}
            size={24}
            color={accepted ? colors.primary : colors.textMuted}
          />
          <Text tone="muted" style={styles.flex}>
            J’ai 18 ans ou plus et je m’engage à rester respectueux·se envers les autres membres.
          </Text>
        </Pressable>
        {touched && !accepted ? <Text tone="danger">Cette confirmation est nécessaire.</Text> : null}
        {register.error ? (
          <Text tone="danger" testID="register-error">
            {describeError(register.error)}
          </Text>
        ) : null}
        <Button label="Continuer" onPress={submit} loading={register.isPending} testID="register-submit" />
        <Button label="J'ai déjà un compte" variant="ghost" size="md" onPress={() => navigation.replace("Login")} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  consent: { flexDirection: "row", gap: 12, alignItems: "flex-start" },
  flex: { flex: 1 }
});
