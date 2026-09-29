import React, { useRef, useState } from "react";
import type { TextInput } from "react-native";
import { changePassword } from "../api/account";
import { errorStatus, messageFromError } from "../lib/errors";
import { validateLoginPassword, validateNewPassword } from "../lib/validation";
import { showToast } from "../shared/feedback/toast";
import { PasswordInput } from "../shared/forms/PasswordInput";
import { Button } from "../shared/ui/Button";
import { Notice } from "../shared/ui/Notice";
import { ProfileSubScreen } from "./ProfileSubScreen";

export function SettingsPasswordScreen({ onBack }: { onBack: () => void }) {
  const newRef = useRef<TextInput>(null);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [currentError, setCurrentError] = useState<string | null>(null);
  const [nextError, setNextError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit() {
    const nextCurrentError = validateLoginPassword(current);
    const newError = validateNewPassword(next, current);
    setCurrentError(nextCurrentError);
    setNextError(newError);
    setServerError(null);
    if (nextCurrentError || newError) {
      return;
    }
    setLoading(true);
    try {
      await changePassword(current, next);
      showToast("Password updated. Other devices were signed out.");
      onBack();
    } catch (error) {
      const status = errorStatus(error);
      const message = messageFromError(error);
      if (status === 401 || status === 403) {
        setCurrentError("Your current password is incorrect.");
      } else if (status === 400 || status === 422) {
        setNextError(message);
      } else {
        setServerError(message);
      }
      setLoading(false);
    }
  }

  return (
    <ProfileSubScreen
      onBack={onBack}
      subtitle="Changing your password signs you out everywhere else."
      testID="settings-password-screen"
      title="Change password"
    >
      <PasswordInput
        autoComplete="current-password"
        editable={!loading}
        error={currentError}
        label="Current password"
        onChangeText={(value) => {
          setCurrent(value);
          setCurrentError(null);
        }}
        onSubmitEditing={() => newRef.current?.focus()}
        returnKeyType="next"
        testID="password-current-input"
        value={current}
      />
      <PasswordInput
        autoComplete="new-password"
        editable={!loading}
        error={nextError}
        helperText="At least 8 characters."
        label="New password"
        onChangeText={(value) => {
          setNext(value);
          setNextError(null);
        }}
        onSubmitEditing={() => void submit()}
        ref={newRef}
        returnKeyType="go"
        testID="password-new-input"
        textContentType="newPassword"
        value={next}
      />
      {serverError ? <Notice title={serverError} tone="danger" /> : null}
      <Button label="Update password" loading={loading} onPress={() => void submit()} size="lg" testID="password-submit-button" />
    </ProfileSubScreen>
  );
}
