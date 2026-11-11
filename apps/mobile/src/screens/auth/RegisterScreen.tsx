import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Screen, Text, TextField } from "../../ui";
import { looksLikeEmail, passwordProblem } from "../../utils/validation";

export function RegisterScreen() {
  const { signIn } = useAuth();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(email.trim(), password),
    onSuccess: signIn
  });

  const pwdError = touched ? passwordProblem(password) : null;
  const canSubmit = looksLikeEmail(email) && !passwordProblem(password) && adult;

  return (
    <Screen
      testID="register-screen"
      footer={
        <Button
          testID="register-submit"
          label="Créer mon compte"
          loading={register.isPending}
          disabled={!canSubmit}
          onPress={() => register.mutate()}
        />
      }
    >
      <View style={{ gap: 6 }}>
        <Text variant="title">Bienvenue</Text>
        <Text tone="muted">Quelques secondes suffisent. Tout est gratuit.</Text>
      </View>
      <TextField
        testID="register-email"
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
      />
      <TextField
        testID="register-password"
        label="Mot de passe"
        value={password}
        onChangeText={(v) => {
          setPassword(v);
          setTouched(true);
        }}
        secureTextEntry
        autoComplete="new-password"
        textContentType="newPassword"
        error={pwdError}
        hint="8 caractères minimum, avec une lettre et un chiffre."
      />
      <Pressable
        testID="register-adult"
        accessibilityRole="checkbox"
        accessibilityState={{ checked: adult }}
        accessibilityLabel="Je confirme avoir 18 ans ou plus"
        onPress={() => setAdult((v) => !v)}
        style={styles.check}
      >
        <Ionicons
          name={adult ? "checkbox" : "square-outline"}
          size={24}
          color={adult ? colors.primary : colors.textMuted}
        />
        <Text style={styles.flex}>Je confirme avoir 18 ans ou plus et m’engage à respecter les autres membres.</Text>
      </Pressable>
      {register.isError ? (
        <Text tone="danger" accessibilityLiveRegion="polite">
          {errorMessage(register.error)}
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  check: { flexDirection: "row", gap: 12, alignItems: "flex-start" },
  flex: { flex: 1 }
});
