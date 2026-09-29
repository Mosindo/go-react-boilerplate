import React, { useRef, useState } from "react";
import type { TextInput } from "react-native";
import { useAuth } from "../hooks/useAuth";
import { validateBirthDate, maskBirthDateInput } from "../lib/dates";
import { errorStatus, messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { normalizeEmail, validateEmail, validatePassword } from "../lib/validation";
import { PasswordInput } from "../shared/forms/PasswordInput";
import { Button } from "../shared/ui/Button";
import { Input } from "../shared/ui/Input";
import { Notice } from "../shared/ui/Notice";
import { Text } from "../shared/ui/Text";
import { AuthFormLayout } from "./AuthFormLayout";

type Props = {
  onBack: () => void;
  onLogin: () => void;
};

export function AuthRegisterScreen({ onBack, onLogin }: Props) {
  const { register } = useAuth();
  const passwordRef = useRef<TextInput>(null);
  const birthRef = useRef<TextInput>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [birthDate, setBirthDate] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; birthDate?: string }>(
    {}
  );
  const [serverError, setServerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit() {
    const birth = validateBirthDate(birthDate);
    const nextErrors = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      birthDate: birth.ok ? undefined : birth.error
    };
    setErrors(nextErrors);
    setServerError(null);
    if (nextErrors.email || nextErrors.password || !birth.ok) {
      return;
    }
    setLoading(true);
    try {
      await register(normalizeEmail(email), password, birth.iso);
    } catch (error) {
      const status = errorStatus(error);
      const message = messageFromError(
        error,
        "We could not create your account. Please try again."
      );
      if (status === 409) {
        setErrors({ email: "An account with this email already exists. Try signing in instead." });
      } else if (status === 422) {
        setErrors({ birthDate: message });
      } else {
        setServerError(message);
      }
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout
      onBack={onBack}
      subtitle={strings.auth.registerBody}
      testID="register-screen"
      title={strings.auth.registerTitle}
    >
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        editable={!loading}
        error={errors.email}
        keyboardType="email-address"
        label="Email"
        onChangeText={(value) => {
          setEmail(value);
          setErrors((prev) => ({ ...prev, email: undefined }));
        }}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="register-email-input"
        textContentType="emailAddress"
        value={email}
      />
      <PasswordInput
        autoComplete="new-password"
        editable={!loading}
        error={errors.password}
        helperText={strings.auth.passwordHint}
        label="Password"
        onChangeText={(value) => {
          setPassword(value);
          setErrors((prev) => ({ ...prev, password: undefined }));
        }}
        onSubmitEditing={() => birthRef.current?.focus()}
        ref={passwordRef}
        returnKeyType="next"
        testID="register-password-input"
        textContentType="newPassword"
        value={password}
      />
      <Input
        editable={!loading}
        error={errors.birthDate}
        helperText={strings.auth.birthDateHint}
        keyboardType="number-pad"
        label="Birth date (DD/MM/YYYY)"
        maxLength={10}
        onChangeText={(value) => {
          setBirthDate(maskBirthDateInput(value));
          setErrors((prev) => ({ ...prev, birthDate: undefined }));
        }}
        onSubmitEditing={() => void submit()}
        placeholder="DD/MM/YYYY"
        ref={birthRef}
        returnKeyType="go"
        testID="register-birthdate-input"
        value={birthDate}
      />
      {serverError ? <Notice testID="auth-error" title={serverError} tone="danger" /> : null}
      <Notice title={strings.freePromise} tone="default" />
      <Button
        label={strings.auth.createAccount}
        loading={loading}
        onPress={() => void submit()}
        size="lg"
        testID="register-submit-button"
      />
      <Text tone="muted" variant="caption">
        By creating an account you confirm that you are at least 18 years old.
      </Text>
      <Button
        label="Already have an account? Sign in"
        onPress={onLogin}
        testID="register-switch-login"
        variant="ghost"
      />
    </AuthFormLayout>
  );
}
