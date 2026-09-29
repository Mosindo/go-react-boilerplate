import React, { useRef, useState } from "react";
import type { TextInput } from "react-native";
import { useAuth } from "../hooks/useAuth";
import { messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { normalizeEmail, validateEmail, validateLoginPassword } from "../lib/validation";
import { PasswordInput } from "../shared/forms/PasswordInput";
import { Button } from "../shared/ui/Button";
import { Input } from "../shared/ui/Input";
import { Notice } from "../shared/ui/Notice";
import { AuthFormLayout } from "./AuthFormLayout";

type Props = {
  notice: string | null;
  onBack: () => void;
  onForgot: () => void;
  onRegister: () => void;
};

export function AuthLoginScreen({ notice, onBack, onForgot, onRegister }: Props) {
  const { signIn } = useAuth();
  const passwordRef = useRef<TextInput>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit() {
    const nextEmailError = validateEmail(email);
    const nextPasswordError = validateLoginPassword(password);
    setEmailError(nextEmailError);
    setPasswordError(nextPasswordError);
    setServerError(null);
    if (nextEmailError || nextPasswordError) {
      return;
    }
    setLoading(true);
    try {
      await signIn(normalizeEmail(email), password);
    } catch (error) {
      setServerError(messageFromError(error, "We could not sign you in. Please try again."));
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout
      onBack={onBack}
      subtitle={strings.auth.loginBody}
      testID="login-screen"
      title={strings.auth.loginTitle}
    >
      {notice ? <Notice title={notice} tone="success" /> : null}
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        editable={!loading}
        error={emailError}
        keyboardType="email-address"
        label="Email"
        onChangeText={(value) => {
          setEmail(value);
          setEmailError(null);
        }}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="auth-email-input"
        textContentType="emailAddress"
        value={email}
      />
      <PasswordInput
        autoComplete="current-password"
        editable={!loading}
        error={passwordError}
        label="Password"
        onChangeText={(value) => {
          setPassword(value);
          setPasswordError(null);
        }}
        onSubmitEditing={() => void submit()}
        ref={passwordRef}
        returnKeyType="go"
        testID="auth-password-input"
        value={password}
      />
      {serverError ? <Notice testID="auth-error" title={serverError} tone="danger" /> : null}
      <Button
        label={strings.auth.signIn}
        loading={loading}
        onPress={() => void submit()}
        size="lg"
        testID="auth-submit-button"
      />
      <Button
        label="Forgot your password?"
        onPress={onForgot}
        testID="auth-forgot-button"
        variant="ghost"
      />
      <Button
        label="New here? Create an account"
        onPress={onRegister}
        testID="auth-switch-register"
        variant="ghost"
      />
    </AuthFormLayout>
  );
}
