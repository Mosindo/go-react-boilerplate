import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { deleteAccount } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import { ConfirmDialog, showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Card, Input, Text } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

export default function DeleteAccountScreen() {
  const { forgetSession } = useAuth();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState(false);
  const mutation = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      setConfirm(false);
      await forgetSession();
      showToast("Ton compte et tes données ont été supprimés.");
    },
    onError: () => setConfirm(false)
  });
  return (
    <Screen testID="delete-account-screen">
      <ScreenHeader title="Supprimer mon compte" />
      <Card>
        <Text weight="semibold">Cette action est définitive</Text>
        <Text tone="muted" variant="label">
          Ton profil, tes photos, tes matchs, tes conversations et tes notifications seront supprimés immédiatement. Les signalements que tu
          as faits sont conservés de façon anonyme pour la sécurité des membres.
        </Text>
      </Card>
      <Input
        autoComplete="current-password"
        error={mutation.isError ? errorMessage(mutation.error) : null}
        label="Confirme avec ton mot de passe"
        onChangeText={setPassword}
        secureTextEntry
        testID="delete-password"
        value={password}
      />
      <Button
        disabled={!password}
        label="Supprimer définitivement"
        onPress={() => setConfirm(true)}
        testID="delete-submit"
        variant="danger"
      />
      <ConfirmDialog
        confirmLabel="Supprimer"
        destructive
        loading={mutation.isPending}
        message="Tu ne pourras pas récupérer ton compte."
        onCancel={() => setConfirm(false)}
        onConfirm={() => mutation.mutate()}
        title="Dernière confirmation"
        visible={confirm}
      />
    </Screen>
  );
}
