import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { useAuth } from "../../providers/AuthProvider";
import { Button, Card, Header, Screen, showToast, Text, TextField } from "../../ui";

export function DeleteAccountScreen() {
  const { forgetSession } = useAuth();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      await forgetSession();
      showToast("Ton compte et toutes tes données ont été supprimés.", "success");
    }
  });
  return (
    <Screen keyboard scroll header={<Header back title="Supprimer mon compte" />}>
      <Card>
        <Text variant="bodyStrong">Cette action est définitive.</Text>
        <Text tone="muted">
          Ton profil, tes photos, tes likes, tes matchs et tes conversations seront supprimés immédiatement et ne pourront pas être récupérés.
        </Text>
      </Card>
      <TextField label="Confirme avec ton mot de passe" onChangeText={setPassword} secureTextEntry testID="delete-password" value={password} />
      {remove.isError ? (
        <Text tone="danger">
          {remove.error && "code" in remove.error && remove.error.code === "invalid_credentials" ? "Mot de passe incorrect." : errorMessage(remove.error)}
        </Text>
      ) : null}
      <Button
        disabled={!password}
        label="Supprimer définitivement"
        loading={remove.isPending}
        onPress={() => remove.mutate()}
        testID="delete-confirm"
        variant="danger"
      />
    </Screen>
  );
}
