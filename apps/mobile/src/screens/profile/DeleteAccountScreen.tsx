import React, { useState } from "react";
import { deleteAccount } from "../../api/auth";
import { useAuth } from "../../hooks/useAuth";
import { confirmAction } from "../../lib/platform";
import { showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, FormField, Input, Notice, Text } from "../../shared/ui";

export function DeleteAccountScreen() {
  const { accountDeleted } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!password) return setError("Saisissez votre mot de passe pour confirmer.");
    if (
      !(await confirmAction(
        "Supprimer définitivement votre compte ?",
        "Profil, photos, matchs et messages seront effacés. Cette action est irréversible.",
        "Supprimer mon compte"
      ))
    )
      return;
    setBusy(true);
    setError(null);
    try {
      await deleteAccount(password);
      await accountDeleted();
      showToast("Votre compte et vos données ont été supprimés.", "success");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Suppression impossible.");
      setBusy(false);
    }
  };

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="delete-account-screen">
      <Text>
        La suppression est immédiate et définitive : votre profil, vos photos, vos matchs, vos conversations et vos signalements sont
        effacés de nos serveurs.
      </Text>
      {error ? <Notice message={error} /> : null}
      <FormField label="Mot de passe">
        <Input testID="delete-password" value={password} onChangeText={setPassword} secureTextEntry autoCapitalize="none" />
      </FormField>
      <Button testID="delete-submit" label="Supprimer mon compte" variant="danger" loading={busy} onPress={() => void submit()} />
    </ScreenContainer>
  );
}
