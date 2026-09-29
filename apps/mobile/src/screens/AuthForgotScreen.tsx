import React, { useState } from "react";
import { requestPasswordReset } from "../api/auth";
import { messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { normalizeEmail, validateEmail } from "../lib/validation";
import { Button } from "../shared/ui/Button";
import { Input } from "../shared/ui/Input";
import { Notice } from "../shared/ui/Notice";
import { AuthFormLayout } from "./AuthFormLayout";

type Props = {
  onBack: () => void;
  onHaveToken: () => void;
};

export function AuthForgotScreen({ onBack, onHaveToken }: Props) {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function submit() {
    const emailError = validateEmail(email);
    setError(emailError);
    setServerError(null);
    if (emailError) {
      return;
    }
    setLoading(true);
    try {
      await requestPasswordReset(normalizeEmail(email));
      setSent(true);
    } catch (err) {
      setServerError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout
      onBack={onBack}
      subtitle={strings.auth.forgotBody}
      testID="forgot-screen"
      title={strings.auth.forgotTitle}
    >
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        editable={!loading}
        error={error}
        keyboardType="email-address"
        label="Email"
        onChangeText={(value) => {
          setEmail(value);
          setError(null);
        }}
        onSubmitEditing={() => void submit()}
        returnKeyType="send"
        testID="forgot-email-input"
        textContentType="emailAddress"
        value={email}
      />
      {sent ? <Notice testID="forgot-sent" title={strings.auth.forgotSent} tone="success" /> : null}
      {serverError ? <Notice title={serverError} tone="danger" /> : null}
      <Button
        label={sent ? "Send again" : "Send reset token"}
        loading={loading}
        onPress={() => void submit()}
        size="lg"
        testID="forgot-submit-button"
        variant={sent ? "outline" : "primary"}
      />
      <Button
        label="I already have a token"
        onPress={onHaveToken}
        testID="forgot-have-token"
        variant={sent ? "primary" : "ghost"}
      />
    </AuthFormLayout>
  );
}
