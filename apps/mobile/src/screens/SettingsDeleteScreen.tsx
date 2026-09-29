import React, { useState } from "react";
import { useAuth } from "../hooks/useAuth";
import { errorStatus, messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { validateLoginPassword } from "../lib/validation";
import { PasswordInput } from "../shared/forms/PasswordInput";
import { Button } from "../shared/ui/Button";
import { ConfirmSheet } from "../shared/ui/ConfirmSheet";
import { Notice } from "../shared/ui/Notice";
import { ProfileSubScreen } from "./ProfileSubScreen";

type Stage = "explain" | "password";

/** Account deletion with a double confirmation and the account password. */
export function SettingsDeleteScreen({ onBack }: { onBack: () => void }) {
  const { deleteAccount } = useAuth();
  const [stage, setStage] = useState<Stage>("explain");
  const [firstConfirm, setFirstConfirm] = useState(false);
  const [finalConfirm, setFinalConfirm] = useState(false);
  const [password, setPassword] = useState("");
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function requestFinal() {
    const error = validateLoginPassword(password);
    setPasswordError(error);
    if (!error) {
      setFinalConfirm(true);
    }
  }

  async function performDelete() {
    setLoading(true);
    setServerError(null);
    try {
      await deleteAccount(password);
    } catch (error) {
      const status = errorStatus(error);
      setFinalConfirm(false);
      if (status === 401 || status === 403) {
        setPasswordError("That password is incorrect.");
      } else {
        setServerError(messageFromError(error));
      }
      setLoading(false);
    }
  }

  return (
    <ProfileSubScreen onBack={onBack} testID="settings-delete-screen" title="Delete account">
      <Notice description={strings.profile.deleteExplain} title="This is permanent" tone="danger" />

      {stage === "explain" ? (
        <Button
          label="I want to delete my account"
          onPress={() => setFirstConfirm(true)}
          size="lg"
          testID="delete-start-button"
          variant="destructive"
        />
      ) : (
        <>
          <PasswordInput
            autoComplete="current-password"
            editable={!loading}
            error={passwordError}
            helperText="Enter your password to confirm it is really you."
            label="Password"
            onChangeText={(value) => {
              setPassword(value);
              setPasswordError(null);
            }}
            returnKeyType="go"
            testID="delete-password-input"
            value={password}
          />
          {serverError ? <Notice title={serverError} tone="danger" /> : null}
          <Button
            label="Delete my account forever"
            onPress={requestFinal}
            size="lg"
            testID="delete-confirm-button"
            variant="destructive"
          />
        </>
      )}
      <Button label="Keep my account" onPress={onBack} variant="ghost" />

      <ConfirmSheet
        confirmLabel="Yes, continue"
        destructive
        message="Your profile, photos, matches and conversations will be erased and cannot be recovered. Do you want to continue?"
        onCancel={() => setFirstConfirm(false)}
        onConfirm={() => {
          setFirstConfirm(false);
          setStage("password");
        }}
        title="Delete your account?"
        visible={firstConfirm}
      />
      <ConfirmSheet
        confirmLabel="Delete everything"
        destructive
        loading={loading}
        message="This is your last chance. Everything tied to your account will be erased immediately."
        onCancel={() => setFinalConfirm(false)}
        onConfirm={() => void performDelete()}
        testID="delete-final-sheet"
        title="Delete forever?"
        visible={finalConfirm}
      />
    </ProfileSubScreen>
  );
}
