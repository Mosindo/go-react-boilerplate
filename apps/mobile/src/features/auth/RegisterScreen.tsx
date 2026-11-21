import React, { useState } from "react";
import { View } from "react-native";
import { errorMessage } from "../../api/client";
import { makeStyles } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/format";
import { useAuth } from "./AuthProvider";

export function RegisterScreen() {
  const styles = useStyles();
  const { signUp } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    const next: typeof errors = {};
    if (!isValidEmail(email)) next.email = "Adresse e-mail invalide.";
    const pwd = passwordProblem(password);
    if (pwd) next.password = pwd;
    setErrors(next);
    if (Object.keys(next).length > 0) return;
    setLoading(true);
    try {
      await signUp(email, password);
    } catch (err) {
      setErrors({ form: errorMessage(err) });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen
      scroll
      keyboard
      footer={
        <>
          {errors.form ? (
            <Text variant="caption" tone="danger" align="center" accessibilityLiveRegion="polite">
              {errors.form}
            </Text>
          ) : null}
          <Button testID="register-submit" label="Créer mon compte" size="lg" loading={loading} onPress={submit} />
          <Text variant="caption" tone="subtle" align="center">
            En continuant, vous confirmez avoir au moins 18 ans.
          </Text>
        </>
      }
    >
      <Header back />
      <Text variant="title" accessibilityRole="header">
        Créons votre compte
      </Text>
      <Text tone="muted">Il vous faudra ensuite une minute pour présenter qui vous êtes.</Text>
      <View style={styles.form}>
        <TextField
          testID="register-email"
          label="E-mail"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          textContentType="emailAddress"
          placeholder="vous@exemple.fr"
          error={errors.email}
        />
        <TextField
          testID="register-password"
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="new-password"
          textContentType="newPassword"
          hint="8 caractères minimum, avec une lettre et un chiffre."
          error={errors.password}
          onSubmitEditing={submit}
        />
      </View>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  form: { gap: t.spacing.lg, marginTop: t.spacing.sm }
}));
