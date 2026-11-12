import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button, Header, Icon, InlineError, Screen, Text, TextField } from "../../ui";
import { validateEmail, validatePassword } from "../../utils/format";

export default function RegisterScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Register">) {
  const { signIn } = useAuth();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const emailError = touched ? validateEmail(email) : null;
  const passwordError = touched ? validatePassword(password) : null;

  const submit = async () => {
    setTouched(true);
    if (validateEmail(email) || validatePassword(password)) return;
    if (!adult) {
      setError("Vous devez confirmer avoir 18 ans ou plus.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(await authApi.register(email.trim(), password));
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll keyboard padded={false}>
      <Header title="" onBack={navigation.goBack} />
      <View style={styles.body}>
        <Text variant="title" serif accessibilityRole="header">
          Créer votre compte
        </Text>
        <Text tone="muted">Votre email reste privé : il n’est jamais montré aux autres membres.</Text>
        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          error={emailError}
          testID="register-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec une lettre et un chiffre."
          error={passwordError}
          testID="register-password"
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: adult }}
          onPress={() => setAdult((v) => !v)}
          style={styles.check}
          testID="register-adult"
        >
          <View style={[styles.box, { borderColor: adult ? colors.primary : colors.borderStrong, backgroundColor: adult ? colors.primary : "transparent" }]}>
            {adult ? <Icon name="checkmark" size={16} color={colors.onPrimary} /> : null}
          </View>
          <Text style={styles.checkText}>J’ai 18 ans ou plus et je m’engage à respecter les autres membres.</Text>
        </Pressable>
        <InlineError message={error} />
        <Button label="Continuer" onPress={submit} loading={busy} testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { paddingHorizontal: spacing.lg, gap: spacing.md },
  check: { flexDirection: "row", gap: spacing.sm, alignItems: "center", paddingVertical: spacing.xs },
  box: { width: 24, height: 24, borderRadius: 7, borderWidth: 2, alignItems: "center", justifyContent: "center" },
  checkText: { flex: 1 }
});
