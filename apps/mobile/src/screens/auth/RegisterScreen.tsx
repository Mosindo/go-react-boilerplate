import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { register } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Header, Icon, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Register">;

export function RegisterScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accepted, setAccepted] = useState(false);
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const pwdError = touched ? passwordProblem(password) : null;

  const submit = async () => {
    setTouched(true);
    if (!isValidEmail(email) || passwordProblem(password)) return;
    if (!accepted) {
      setError("Vous devez confirmer avoir 18 ans et accepter les conditions.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await signIn(await register(email.trim(), password, accepted));
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen
      edges={["top", "bottom"]}
      footer={<Button label="Créer mon compte" loading={loading} onPress={submit} testID="register-submit" />}
      scroll
      testID="register-screen"
    >
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Créez votre compte</Text>
        <Text tone="muted">Gratuit, aujourd’hui comme demain. Votre e-mail n’est jamais visible par les autres membres.</Text>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          placeholder="vous@exemple.fr"
          testID="register-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoComplete="new-password"
          error={pwdError}
          hint="8 caractères minimum, dont une lettre et un chiffre."
          label="Mot de passe"
          onChangeText={setPassword}
          secureToggle
          testID="register-password"
          textContentType="newPassword"
          value={password}
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: accepted }}
          onPress={() => setAccepted((v) => !v)}
          style={styles.checkRow}
          testID="register-terms"
        >
          <View
            style={[
              styles.box,
              { borderColor: accepted ? colors.primary : colors.border, backgroundColor: accepted ? colors.primary : colors.surface }
            ]}
          >
            {accepted ? <Icon color={colors.onPrimary} name="checkmark" size={16} /> : null}
          </View>
          <Text style={styles.checkText} variant="caption">
            J’ai 18 ans ou plus, j’accepte les conditions d’utilisation et je m’engage à respecter les autres membres.
          </Text>
        </Pressable>
        {error ? (
          <Text accessibilityLiveRegion="polite" testID="register-error" tone="danger">
            {error}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingTop: spacing.md },
  checkRow: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  box: { width: 24, height: 24, borderRadius: radii.sm - 4, borderWidth: 2, alignItems: "center", justifyContent: "center", marginTop: 2 },
  checkText: { flex: 1 }
});
