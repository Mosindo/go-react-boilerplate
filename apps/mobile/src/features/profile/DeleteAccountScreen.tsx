import React, { useState } from "react";
import { deleteAccount } from "../../api/auth";
import { getErrorMessage, isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { Button, ConfirmDialog, Input, Screen, Text, toast } from "../../ui";

export function DeleteAccountScreen() {
  const { finishAccountDeletion } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  function ask() {
    if (!password) {
      setError("Enter your password to continue.");
      return;
    }
    setError(null);
    setConfirming(true);
  }

  async function doDelete() {
    setBusy(true);
    try {
      await deleteAccount(password);
      setConfirming(false);
      finishAccountDeletion();
      toast.info("Your account was deleted.");
    } catch (e) {
      setConfirming(false);
      setBusy(false);
      setError(
        isApiError(e) && (e.status === 401 || e.status === 403) ? "That password isn't correct." : getErrorMessage(e)
      );
    }
  }

  return (
    <Screen keyboardAvoiding scroll>
      <Text variant="title">Delete your account</Text>
      <Text tone="muted">
        This permanently removes your profile, photos, matches and messages. It cannot be undone. If you just need a
        break, you can pause your profile in Discovery preferences instead.
      </Text>
      <Input
        autoCapitalize="none"
        error={error}
        label="Confirm with your password"
        onChangeText={setPassword}
        passwordToggle
        secureTextEntry
        value={password}
      />
      <Button label="Delete my account" onPress={ask} variant="danger" />
      <ConfirmDialog
        confirmLabel="Delete forever"
        destructive
        loading={busy}
        message="All your data will be erased. This cannot be undone."
        onCancel={() => setConfirming(false)}
        onConfirm={doDelete}
        title="Delete your account?"
        visible={confirming}
      />
    </Screen>
  );
}
