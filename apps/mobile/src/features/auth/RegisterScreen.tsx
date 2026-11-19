import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/tokens";
import { Button, Header, Icon, Screen, Text, TextField } from "../../ui";
import { isValidEmail, normalizeEmail, passwordProblem } from "../../lib/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Register">;

export function RegisterScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register(normalizeEmail(email), password),
    onSuccess: signIn
  });

  const emailError = submitted && !isValidEmail(email) ? "Saisissez une adresse e-mail valide." : null;
  const pwdProblem = passwordProblem(password);
  const passwordError = submitted ? pwdProblem : null;
  const adultError = submitted && !adult;

  const submit = () => {
    setSubmitted(true);
    if (!isValidEmail(email) || pwdProblem || !adult) return;
    register.mutate();
  };

  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Créer votre compte</Text>
        <Text tone="muted">Gratuit, sans abonnement ni option payante. Votre e-mail n’est jamais visible.</Text>
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
          <Icon name={adult ? "checkbox" : "square-outline"} size={24} color={adultError ? colors.danger : colors.primary} />
          <Text style={styles.checkText} tone={adultError ? "danger" : "default"}>
            J’ai 18 ans ou plus et je m’engage à respecter les autres membres.
          </Text>
        </Pressable>
        {register.error ? (
          <Text tone="danger" accessibilityLiveRegion="polite">
            {errorMessage(register.error)}
          </Text>
        ) : null}
        <Button label="Continuer" onPress={submit} loading={register.isPending} testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, paddingTop: spacing.md },
  check: { flexDirection: "row", gap: spacing.sm, alignItems: "center" },
  checkText: { flex: 1 }
});
