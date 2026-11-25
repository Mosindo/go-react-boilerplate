import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { useSession } from "../../hooks/useSession";
import type { AuthScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { normalizeEmail, validateEmail, validatePassword } from "../../utils/validation";

export function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [adult, setAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const errors = {
    email: validateEmail(email),
    password: validatePassword(password),
    confirm: confirm !== password ? "Les mots de passe ne correspondent pas." : null,
    adult: adult ? null : "Vous devez confirmer avoir 18 ans ou plus."
  };
  const valid = !errors.email && !errors.password && !errors.confirm && !errors.adult;

  const submit = () => {
    setSubmitted(true);
    if (valid) {
      register.mutate();
    }
  };

  return (
    <Screen scroll testID="register-screen">
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Créer votre compte</Text>
        <Text variant="small" color="textMuted">
          Votre e-mail reste privé : il n’est jamais montré aux autres membres.
        </Text>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          error={submitted ? errors.email : null}
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
          error={submitted ? errors.password : null}
          testID="register-password"
        />
        <TextField
          label="Confirmer le mot de passe"
          value={confirm}
          onChangeText={setConfirm}
          secureTextEntry
          autoComplete="new-password"
          error={submitted ? errors.confirm : null}
          onSubmitEditing={submit}
          testID="register-confirm"
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: adult }}
          onPress={() => setAdult((v) => !v)}
          style={styles.check}
          testID="register-adult"
        >
          <Ionicons name={adult ? "checkbox" : "square-outline"} size={22} color={adult ? colors.primary : colors.textSubtle} />
          <Text variant="small" color="textMuted" style={styles.flex}>
            J’ai 18 ans ou plus et je m’engage à respecter les autres membres.
          </Text>
        </Pressable>
        {submitted && errors.adult ? (
          <Text variant="caption" color="danger">
            {errors.adult}
          </Text>
        ) : null}
        {register.error ? (
          <Text variant="small" color="danger" testID="register-error">
            {errorMessage(register.error)}
          </Text>
        ) : null}
        <Button label="Continuer" onPress={submit} loading={register.isPending} testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md,
    paddingTop: spacing.md
  },
  check: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  flex: {
    flex: 1
  }
});
