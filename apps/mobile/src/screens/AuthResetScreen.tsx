import React, { useRef, useState } from "react";
import type { TextInput } from "react-native";
import { resetPassword } from "../api/auth";
import { errorStatus, messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { validateNewPassword } from "../lib/validation";
import { PasswordInput } from "../shared/forms/PasswordInput";
import { Button } from "../shared/ui/Button";
import { Input } from "../shared/ui/Input";
import { Notice } from "../shared/ui/Notice";
import { AuthFormLayout } from "./AuthFormLayout";

type Props = {
  onBack: () => void;
  onDone: () => void;
};

export function AuthResetScreen({ onBack, onDone }: Props) {
  const passwordRef = useRef<TextInput>(null);
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [tokenError, setTokenError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit() {
    const cleanToken = token.trim();
    const nextTokenError = cleanToken ? null : "Paste the token from your email.";
    const nextPasswordError = validateNewPassword(password);
    setTokenError(nextTokenError);
    setPasswordError(nextPasswordError);
    setServerError(null);
    if (nextTokenError || nextPasswordError) {
      return;
    }
    setLoading(true);
    try {
      await resetPassword(cleanToken, password);
      onDone();
    } catch (err) {
      const status = errorStatus(err);
      if (status === 400 || status === 404 || status === 422) {
        setTokenError("That token is invalid or has expired. Request a new one.");
      } else {
        setServerError(messageFromError(err));
      }
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout
      onBack={onBack}
      subtitle={strings.auth.resetBody}
      testID="reset-screen"
      title={strings.auth.resetTitle}
    >
      <Input
        autoCapitalize="none"
        autoCorrect={false}
        editable={!loading}
        error={tokenError}
        label="Reset token"
        onChangeText={(value) => {
          setToken(value);
          setTokenError(null);
        }}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="reset-token-input"
        value={token}
      />
      <PasswordInput
        autoComplete="new-password"
        editable={!loading}
        error={passwordError}
        helperText={strings.auth.passwordHint}
        label="New password"
        onChangeText={(value) => {
          setPassword(value);
          setPasswordError(null);
        }}
        onSubmitEditing={() => void submit()}
        ref={passwordRef}
        returnKeyType="go"
        testID="reset-password-input"
        textContentType="newPassword"
        value={password}
      />
      {serverError ? <Notice title={serverError} tone="danger" /> : null}
      <Button
        label="Update password"
        loading={loading}
        onPress={() => void submit()}
        size="lg"
        testID="reset-submit-button"
      />
    </AuthFormLayout>
  );
}
