import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import { normalizeEmail, parseBirthDate, validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, FormField, Input, Notice, Text, spacing } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParamList, "Register">;

export default function RegisterScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: (birthDate: string) => authApi.register(normalizeEmail(email), password, birthDate),
    onSuccess: signIn,
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) setError("Un compte existe déjà avec cet e-mail.");
      else if (err instanceof ApiError && err.status === 403) setError("Alma est réservée aux personnes majeures (18 ans et plus).");
      else setError(err instanceof Error ? err.message : "Inscription impossible.");
    }
  });

  const submit = () => {
    const birth = parseBirthDate(day, month, year);
    const message = validateEmail(email) ?? validatePassword(password) ?? (birth.ok ? null : birth.error);
    if (message || !birth.ok) {
      setError(message);
      return;
    }
    setError(null);
    mutation.mutate(birth.iso);
  };

  return (
    <AuthLayout subtitle="Gratuit, sans abonnement ni limite de likes." title="Créer un compte">
      <Input
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        label="E-mail"
        onChangeText={setEmail}
        testID="register-email"
        value={email}
      />
      <Input
        autoComplete="new-password"
        helperText="8 caractères minimum"
        label="Mot de passe"
        onChangeText={setPassword}
        secureTextEntry
        testID="register-password"
        value={password}
      />
      <FormField
        label="Date de naissance"
        helperText="Vous devez avoir 18 ans ou plus. Votre date de naissance n'est jamais affichée, seulement votre âge."
      >
        <View style={styles.dateRow}>
          <Input
            containerStyle={styles.day}
            keyboardType="number-pad"
            maxLength={2}
            onChangeText={setDay}
            placeholder="JJ"
            testID="register-day"
            value={day}
          />
          <Input
            containerStyle={styles.day}
            keyboardType="number-pad"
            maxLength={2}
            onChangeText={setMonth}
            placeholder="MM"
            testID="register-month"
            value={month}
          />
          <Input
            containerStyle={styles.year}
            keyboardType="number-pad"
            maxLength={4}
            onChangeText={setYear}
            placeholder="AAAA"
            testID="register-year"
            value={year}
          />
        </View>
      </FormField>
      {error ? <Notice title={error} tone="danger" /> : null}
      <Button label="Créer mon compte" loading={mutation.isPending} onPress={submit} testID="register-submit" />
      <Text tone="muted" variant="caption">
        En créant un compte, vous confirmez avoir 18 ans ou plus et acceptez de respecter les autres membres.
      </Text>
      <Button label="J'ai déjà un compte" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </AuthLayout>
  );
}

const styles = StyleSheet.create({
  dateRow: { flexDirection: "row", gap: spacing.sm },
  day: { flex: 1 },
  year: { flex: 2 }
});
