import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { deleteAccount } from "../../api";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { validateRequired } from "../../lib/validation";
import { useSession } from "../../session/SessionProvider";
import { Button, Screen, Text, TextField, useDialog, useToast } from "../../ui";

export default function DeleteAccountScreen() {
  const toast = useToast();
  const dialog = useDialog();
  const { endLocalSession } = useSession();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const remove = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      await endLocalSession();
      toast.success(t.deleteAccount.done);
    },
    onError: (err) => toast.error(errorMessage(err))
  });

  const submit = () => {
    const problem = validateRequired(password);
    setError(problem);
    if (problem) return;
    dialog.confirm({
      title: t.deleteAccount.title,
      message: t.deleteAccount.warning,
      confirmLabel: t.deleteAccount.cta,
      destructive: true,
      onConfirm: () => remove.mutate()
    });
  };

  return (
    <Screen scroll testID="delete-account-screen">
      <Text tone="muted">{t.deleteAccount.warning}</Text>
      <TextField
        autoCapitalize="none"
        error={error}
        label={t.deleteAccount.password}
        onChangeText={setPassword}
        secureTextEntry
        testID="delete-password-input"
        value={password}
      />
      <Button
        label={t.deleteAccount.cta}
        loading={remove.isPending}
        onPress={submit}
        testID="delete-submit-button"
        variant="danger"
      />
    </Screen>
  );
}
