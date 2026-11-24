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
import { isValidEmail } from "../../utils/validation";

export default function SignInScreen() {
  const navigation = useNavigation();
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!isValidEmail(email) || !password) {
      setError("Saisis ton e-mail et ton mot de passe.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setSubmitting(false);
    }
  };

  return (
    <Screen scroll testID="signin-screen">
      <Header title="Connexion" onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <TextField
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          testID="signin-email"
        />
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="current-password"
          textContentType="password"
          onSubmitEditing={submit}
          testID="signin-password"
        />
        {error ? (
          <AppText color={colors.danger} accessibilityLiveRegion="polite" testID="signin-error">
            {error}
          </AppText>
        ) : null}
        <Button label="Se connecter" onPress={submit} loading={submitting} testID="signin-submit" />
        <Button label="Mot de passe oublié ?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword", { email })} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, marginTop: spacing.md }
});
