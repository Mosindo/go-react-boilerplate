import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { AppText, Banner, Button, Card, Screen, TextField } from "../../components/ui";
import { Sheet } from "../../components/Modals";
import { authApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { useAuth } from "../../auth/AuthContext";
import { validatePassword } from "../../utils/validation";
import { spacing } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";

export default function SettingsScreen({ navigation }: NativeStackScreenProps<MainStackParams, "Settings">) {
  const { user, signOut, deleteAccount } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [notice, setNotice] = useState<{ tone: "error" | "info"; text: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const changePassword = async () => {
    const problem = validatePassword(next);
    if (!current) return setNotice({ tone: "error", text: "Saisissez votre mot de passe actuel." });
    if (problem) return setNotice({ tone: "error", text: problem });
    setBusy(true);
    try {
      await authApi.changePassword(current, next);
      setCurrent("");
      setNext("");
      setNotice({ tone: "info", text: "Mot de passe modifié. Vos autres appareils ont été déconnectés." });
    } catch (e) {
      setNotice({ tone: "error", text: errorMessage(e) });
    } finally {
      setBusy(false);
    }
  };

  const confirmDelete = async () => {
    setBusy(true);
    setDeleteError(null);
    try {
      await deleteAccount(deletePassword);
    } catch (e) {
      setDeleteError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <AppText variant="muted">Connecté avec {user?.email}</AppText>
      <Button
        testID="blocked-list"
        title="Personnes bloquées"
        variant="secondary"
        onPress={() => navigation.navigate("Blocked")}
      />
      <Card>
        <AppText variant="heading">Changer le mot de passe</AppText>
        {notice ? <Banner tone={notice.tone} message={notice.text} /> : null}
        <TextField
          label="Mot de passe actuel"
          value={current}
          onChangeText={setCurrent}
          secureTextEntry
          autoCapitalize="none"
        />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          autoCapitalize="none"
          hint="8 caractères minimum."
        />
        <Button title="Mettre à jour" variant="secondary" loading={busy} onPress={changePassword} />
      </Card>
      <Card>
        <AppText variant="heading">Vos données</AppText>
        <AppText variant="muted">
          Votre position est enregistrée avec une précision d&apos;environ 1 km et n&apos;est jamais affichée
          : les autres voient seulement une distance approximative. Vos photos sont stockées sur nos serveurs
          et ne sont accessibles que via des liens temporaires.
        </AppText>
      </Card>
      <View style={{ gap: spacing.sm }}>
        <Button testID="logout" title="Se déconnecter" variant="secondary" onPress={() => void signOut()} />
        <Button
          testID="delete-account"
          title="Supprimer mon compte"
          variant="danger"
          onPress={() => setDeleting(true)}
        />
      </View>
      <Sheet visible={deleting} onClose={() => setDeleting(false)}>
        <AppText variant="heading">Supprimer définitivement ?</AppText>
        <AppText variant="muted">
          Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.
        </AppText>
        {deleteError ? <Banner message={deleteError} /> : null}
        <TextField
          testID="delete-password"
          label="Confirmez avec votre mot de passe"
          value={deletePassword}
          onChangeText={setDeletePassword}
          secureTextEntry
          autoCapitalize="none"
        />
        <Button
          testID="confirm-delete"
          title="Supprimer mon compte"
          variant="danger"
          loading={busy}
          disabled={!deletePassword}
          onPress={confirmDelete}
        />
        <Button title="Annuler" variant="ghost" onPress={() => setDeleting(false)} />
      </Sheet>
    </Screen>
  );
}
