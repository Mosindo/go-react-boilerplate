import React, { useRef, useState } from "react";
import { Pressable, StyleSheet, View, type TextInput } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import { isValidEmail, passwordProblem } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { useSession } from "../../session/SessionProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Header, Icon, InlineError, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Register">;

export function RegisterScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const passwordError = touched ? passwordProblem(password) : null;

  const submit = async () => {
    setTouched(true);
    if (!isValidEmail(email) || passwordProblem(password)) return;
    if (!adult) {
      setError("Confirmez que vous avez 18 ans ou plus et acceptez les règles de la communauté.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await signIn(await authApi.register(email.trim(), password));
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="register-screen" edges={["top", "bottom"]}>
      <Header onBack={() => navigation.goBack()} />
      <Text variant="title" style={styles.title} accessibilityRole="header">
        Créer votre compte
      </Text>
      <Text color="textMuted" style={styles.subtitle}>
        Votre e-mail reste privé : il n’est jamais montré aux autres membres.
      </Text>
      <View style={styles.form}>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          returnKeyType="next"
          onSubmitEditing={() => passwordRef.current?.focus()}
          error={emailError}
          testID="register-email"
        />
        <TextField
          ref={passwordRef}
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          error={passwordError}
          testID="register-password"
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: adult }}
          onPress={() => setAdult((v) => !v)}
          style={[styles.check, { borderColor: adult ? colors.primary : colors.border, backgroundColor: colors.surface }]}
          testID="register-adult"
        >
          <View
            style={[
              styles.box,
              { borderColor: adult ? colors.primary : colors.textSubtle, backgroundColor: adult ? colors.primary : "transparent" }
            ]}
          >
            {adult ? <Icon name="checkmark" size={16} color="onPrimary" /> : null}
          </View>
          <Text variant="caption" style={styles.checkText}>
            J’ai 18 ans ou plus et je m’engage à respecter les autres membres : pas de harcèlement, pas de faux profil, pas de contenu
            inapproprié.
          </Text>
        </Pressable>
        <InlineError message={error} />
        <Button label="Continuer" onPress={submit} loading={loading} testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { marginTop: spacing.md },
  subtitle: { marginTop: spacing.xs, marginBottom: spacing.xl },
  form: { gap: spacing.md },
  check: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start", padding: spacing.md, borderRadius: radii.md, borderWidth: 1.5 },
  box: { width: 24, height: 24, borderRadius: 7, borderWidth: 2, alignItems: "center", justifyContent: "center" },
  checkText: { flex: 1 }
});
