import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { isValidEmail, passwordProblem } from "../../utils/validation";

export default function SignUpScreen() {
  const navigation = useNavigation();
  const { signUp } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const emailError = touched && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const passwordError = touched ? passwordProblem(password) : null;

  const submit = async () => {
    setTouched(true);
    if (!isValidEmail(email) || passwordProblem(password)) return;
    setSubmitting(true);
    setError(null);
    try {
      await signUp(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setSubmitting(false);
    }
  };

  return (
    <Screen scroll testID="signup-screen">
      <Header title="Créer un compte" onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <AppText color={colors.textMuted}>Ton e-mail reste privé : il n’apparaît jamais sur ton profil.</AppText>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          error={emailError}
          testID="signup-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="new-password"
          textContentType="newPassword"
          error={passwordError}
          hint="8 caractères minimum, avec une lettre et un chiffre."
          onSubmitEditing={submit}
          testID="signup-password"
        />
        {error ? (
          <AppText color={colors.danger} accessibilityLiveRegion="polite" testID="signup-error">
            {error}
          </AppText>
        ) : null}
        <Button label="Continuer" onPress={submit} loading={submitting} testID="signup-submit" />
        <AppText variant="caption" color={colors.textSubtle} align="center">
          En continuant, tu confirmes avoir au moins 18 ans.
        </AppText>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, marginTop: spacing.md }
});
