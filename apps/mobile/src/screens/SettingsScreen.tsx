import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { authApi } from "../api/endpoints";
import { errorMessage } from "../hooks/queries";
import { validateNewPassword } from "../lib/validation";
import { useSession } from "../state/session";
import { Button, Input, Screen, Sheet, SheetBody, Text, spacing, useToast } from "../ui";

export default function SettingsScreen() {
  const toast = useToast();
  const { user, logout, endSession } = useSession();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [pwError, setPwError] = useState<string | null>(null);
  const [pwBusy, setPwBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  const changePassword = async () => {
    const invalid = validateNewPassword(next) ?? (current ? null : "Saisissez votre mot de passe actuel.");
    if (invalid) return setPwError(invalid);
    setPwError(null);
    setPwBusy(true);
    try {
      await authApi.changePassword(current, next);
      toast.show("Mot de passe modifié. Reconnectez-vous.");
      await endSession();
    } catch (e) {
      setPwError(errorMessage(e));
      setPwBusy(false);
    }
  };

  const deleteAccount = async () => {
    if (!deletePassword) return setDeleteError("Saisissez votre mot de passe.");
    setDeleteError(null);
    setDeleteBusy(true);
    try {
      await authApi.deleteAccount(deletePassword);
      setConfirming(false);
      await endSession();
    } catch (e) {
      setDeleteError(errorMessage(e));
      setDeleteBusy(false);
    }
  };

  return (
    <Screen scroll testID="settings-screen">
      <View style={styles.content}>
        <Text tone="muted">Connecté avec {user?.email}</Text>

        <Text variant="heading">Changer de mot de passe</Text>
        <Input
          autoCapitalize="none"
          label="Mot de passe actuel"
          onChangeText={setCurrent}
          secureTextEntry
          value={current}
        />
        <Input
          autoCapitalize="none"
          error={pwError}
          hint="10 caractères minimum. Vous serez déconnecté de tous vos appareils."
          label="Nouveau mot de passe"
          onChangeText={setNext}
          secureTextEntry
          value={next}
        />
        <Button label="Modifier le mot de passe" loading={pwBusy} onPress={changePassword} variant="secondary" />

        <Button label="Se déconnecter" onPress={() => void logout()} testID="logout" variant="secondary" />

        <Text style={styles.danger} variant="heading">
          Zone dangereuse
        </Text>
        <Text tone="muted">La suppression est définitive : profil, photos, matchs et messages sont effacés.</Text>
        <Button
          label="Supprimer mon compte"
          onPress={() => setConfirming(true)}
          testID="delete-account"
          variant="danger"
        />
      </View>

      <Sheet onClose={() => setConfirming(false)} title="Supprimer mon compte ?" visible={confirming}>
        <SheetBody>
          <Text tone="muted">Cette action est irréversible. Confirmez avec votre mot de passe.</Text>
          <Input
            autoCapitalize="none"
            error={deleteError}
            label="Mot de passe"
            onChangeText={setDeletePassword}
            secureTextEntry
            testID="delete-password"
            value={deletePassword}
          />
          <Button
            label="Supprimer définitivement"
            loading={deleteBusy}
            onPress={deleteAccount}
            testID="delete-confirm"
            variant="danger"
          />
          <Button label="Annuler" onPress={() => setConfirming(false)} variant="ghost" />
        </SheetBody>
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.lg, paddingTop: spacing.lg },
  danger: { marginTop: spacing.xl }
});
