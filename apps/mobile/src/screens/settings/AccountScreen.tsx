import React, { useState } from "react";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../api/errors";
import { useFeedback } from "../../feedback";
import { useAuth } from "../../hooks/useAuth";
import type { AppScreenProps } from "../../navigation/types";
import { Button, Card, Header, Screen, Text, TextField } from "../../ui";
import { validatePassword } from "../../utils/validation";

export default function AccountScreen({ navigation }: AppScreenProps<"Account">) {
  const { user, forgetSession } = useAuth();
  const { toast, confirm } = useFeedback();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [changing, setChanging] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const changePassword = async () => {
    const weak = validatePassword(next);
    if (weak) {
      setError(weak);
      return;
    }
    setError(null);
    setChanging(true);
    try {
      await authApi.changePassword(current, next);
      setCurrent("");
      setNext("");
      toast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setChanging(false);
    }
  };

  const deleteAccount = async () => {
    if (!deletePassword) {
      toast("Saisissez votre mot de passe pour confirmer.", "error");
      return;
    }
    const ok = await confirm({
      title: "Supprimer définitivement votre compte ?",
      message: "Profil, photos, matchs et messages seront effacés. Cette action est irréversible.",
      confirmLabel: "Supprimer mon compte",
      destructive: true
    });
    if (!ok) {
      return;
    }
    setDeleting(true);
    try {
      await authApi.deleteAccount(deletePassword);
      toast("Votre compte a été supprimé.", "success");
      await forgetSession();
    } catch (e) {
      toast(errorMessage(e, "Mot de passe incorrect ou suppression impossible."), "error");
      setDeleting(false);
    }
  };

  return (
    <Screen keyboard scroll testID="account-screen">
      <Header onBack={() => navigation.goBack()} title="Compte" subtitle={user?.email} />
      <Card>
        <Text variant="heading">Changer de mot de passe</Text>
        <TextField autoComplete="current-password" label="Mot de passe actuel" onChangeText={setCurrent} secureToggle value={current} />
        <TextField
          autoComplete="new-password"
          error={error}
          label="Nouveau mot de passe"
          onChangeText={setNext}
          secureToggle
          value={next}
        />
        <Button disabled={!current || !next} label="Mettre à jour" loading={changing} onPress={changePassword} variant="secondary" />
      </Card>
      <Card>
        <Text variant="heading">Supprimer mon compte</Text>
        <Text color="textMuted" variant="caption">
          Toutes vos données personnelles sont effacées immédiatement : profil, photos, likes, matchs et conversations.
        </Text>
        <TextField label="Mot de passe" onChangeText={setDeletePassword} secureToggle testID="delete-password" value={deletePassword} />
        <Button label="Supprimer mon compte" loading={deleting} onPress={deleteAccount} testID="delete-account" variant="danger" />
      </Card>
    </Screen>
  );
}
