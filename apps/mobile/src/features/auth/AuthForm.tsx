import React, { useRef, useState } from "react";
import { StyleSheet, type TextInput, View } from "react-native";
import { MIN_PASSWORD_LENGTH } from "../../config";
import { spacing } from "../../theme";
import { Button, Text, TextField } from "../../shared/ui";
import { normalizeEmail, validateEmail, validatePassword, validatePasswordConfirmation } from "../../shared/validation";

export type AuthFormValues = { email: string; password: string };

type Props = {
  mode: "login" | "register";
  loading: boolean;
  /** Error returned by the server, shown above the button. */
  submitError?: string | null;
  initialEmail?: string;
  onSubmit: (values: AuthFormValues) => void;
};

export function AuthForm({ mode, loading, submitError, initialEmail = "", onSubmit }: Props) {
  const [email, setEmail] = useState(initialEmail);
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; confirmation?: string }>({});
  const passwordRef = useRef<TextInput>(null);
  const confirmationRef = useRef<TextInput>(null);
  const isRegister = mode === "register";

  const submit = () => {
    const nextErrors = {
      email: validateEmail(email) ?? undefined,
      password:
        (isRegister ? validatePassword(password) : password ? null : "Saisissez votre mot de passe.") ?? undefined,
      confirmation: (isRegister ? validatePasswordConfirmation(password, confirmation) : null) ?? undefined
    };
    setErrors(nextErrors);
    if (nextErrors.email || nextErrors.password || nextErrors.confirmation) {
      return;
    }
    onSubmit({ email: normalizeEmail(email), password });
  };

  return (
    <View style={styles.form}>
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        error={errors.email}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="auth-email-input"
        textContentType="emailAddress"
        value={email}
      />
      <TextField
        autoCapitalize="none"
        autoComplete={isRegister ? "new-password" : "current-password"}
        error={errors.password}
        hint={isRegister ? `Au moins ${MIN_PASSWORD_LENGTH} caractères.` : undefined}
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={isRegister ? () => confirmationRef.current?.focus() : submit}
        ref={passwordRef}
        returnKeyType={isRegister ? "next" : "go"}
        secureTextEntry
        testID="auth-password-input"
        textContentType={isRegister ? "newPassword" : "password"}
        value={password}
      />
      {isRegister ? (
        <TextField
          autoCapitalize="none"
          error={errors.confirmation}
          label="Confirmer le mot de passe"
          onChangeText={setConfirmation}
          onSubmitEditing={submit}
          ref={confirmationRef}
          returnKeyType="go"
          secureTextEntry
          testID="auth-confirm-input"
          value={confirmation}
        />
      ) : null}
      {submitError ? (
        <Text accessibilityLiveRegion="polite" color="danger" testID="auth-error" variant="caption">
          {submitError}
        </Text>
      ) : null}
      <Button
        label={isRegister ? "Créer mon compte" : "Se connecter"}
        loading={loading}
        onPress={submit}
        testID="auth-submit-button"
      />
    </View>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
