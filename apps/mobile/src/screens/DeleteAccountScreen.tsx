import { useState } from "react";
import { useToast } from "../components/feedback";
import { Button, Screen, Text, TextField } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { useAuth } from "../state/auth";
import { spacing } from "../theme";

export function DeleteAccountScreen() {
  const { deleteAccount } = useAuth();
  const toast = useToast();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);

  const confirm = async () => {
    setBusy(true);
    try {
      await deleteAccount(password);
      toast.show("Votre compte a été supprimé.", "success");
    } catch (e) {
      toast.show(errorMessage(e), "error");
      setBusy(false);
    }
  };

  return (
    <Screen edges={["bottom"]} scroll>
      <Text style={{ marginBottom: spacing.md }} variant="heading">
        Cette action est définitive
      </Text>
      <Text muted style={{ marginBottom: spacing.lg }}>
        Votre profil, vos photos, vos matchs, vos conversations et vos préférences seront supprimés immédiatement et
        définitivement.
      </Text>
      <TextField label="Mot de passe" onChangeText={setPassword} secure value={password} />
      <Button
        disabled={!password}
        loading={busy}
        onPress={() => void confirm()}
        title="Supprimer définitivement mon compte"
        variant="danger"
      />
    </Screen>
  );
}
