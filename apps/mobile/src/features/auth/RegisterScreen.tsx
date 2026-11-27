import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Button, Header, Icon, InlineError, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordIssue } from "../../utils/format";
import type { AuthScreenProps } from "../../navigation/types";

export function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const { signIn } = useAuth();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(email.trim().toLowerCase(), password),
    onSuccess: signIn
  });

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const pwdError = touched ? passwordIssue(password) : null;
  const canSubmit = isValidEmail(email) && !passwordIssue(password) && adult;

  const submit = () => {
    setTouched(true);
    if (canSubmit) {
      register.mutate();
    }
  };

  return (
    <Screen scroll keyboard testID="register-screen">
      <Header onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <Text variant="title" accessibilityRole="header">
          Crée ton compte
        </Text>
        <Text tone="muted">Deux minutes, et c’est parti. Aucune carte bancaire, jamais.</Text>
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
          secure
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          error={pwdError}
          testID="register-password"
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: adult }}
          onPress={() => setAdult((v) => !v)}
          style={styles.check}
          testID="register-adult"
        >
          <Icon name={adult ? "checkbox" : "square-outline"} size={24} color={adult ? colors.primary : colors.textMuted} />
          <Text variant="caption" tone="muted" style={styles.flex}>
            J’ai 18 ans ou plus et je m’engage à respecter les autres membres : pas de harcèlement, pas de faux profil, pas de contenu
            inapproprié.
          </Text>
        </Pressable>
        {touched && !adult ? <Text variant="caption" tone="danger">Merci de confirmer ces conditions.</Text> : null}
        <InlineError message={register.error ? errorMessage(register.error) : null} />
        <Button label="Créer mon compte" size="lg" loading={register.isPending} onPress={submit} testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, maxWidth: 480, width: "100%", alignSelf: "center", paddingTop: spacing.md },
  check: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  flex: { flex: 1 }
});
