import React, { useState } from "react";
import { Alert } from "react-native";
import { useAuth } from "../../hooks/useAuth";
import { strings } from "../../i18n/strings";
import { reportError } from "../../shared/feedback";
import { FormScreen } from "../../shared/layout";
import { Button, Input, Notice } from "../../shared/ui";

export default function DeleteAccountScreen() {
  const { deleteAccount } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function submit() {
    if (!password) {
      setError(strings.validation.passwordRequired);
      return;
    }
    setError(null);
    Alert.alert(strings.deleteAccount.confirmTitle, strings.deleteAccount.confirmBody, [
      { text: strings.common.cancel, style: "cancel" },
      {
        text: strings.deleteAccount.confirmAction,
        style: "destructive",
        onPress: () => {
          setBusy(true);
          deleteAccount(password).catch((failure: unknown) => {
            reportError(failure);
            setBusy(false);
          });
        }
      }
    ]);
  }

  return (
    <FormScreen edges={["left", "right", "bottom"]}>
      <Notice
        description={strings.deleteAccount.warningBody}
        title={strings.deleteAccount.warningTitle}
        tone="danger"
      />
      <Input
        autoCapitalize="none"
        error={error}
        label={strings.deleteAccount.passwordLabel}
        onChangeText={setPassword}
        secureTextEntry
        textContentType="password"
        value={password}
      />
      <Button label={strings.deleteAccount.action} loading={busy} onPress={submit} variant="destructive" />
    </FormScreen>
  );
}
