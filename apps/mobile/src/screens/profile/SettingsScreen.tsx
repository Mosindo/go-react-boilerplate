import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { deleteAccount } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { clearLocation, getMyProfile, updatePrivacy } from "../../api/profile";
import { setProfileCache, useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import {
  Button,
  Card,
  ConfirmDialog,
  Dialog,
  Header,
  ListRow,
  Screen,
  SectionTitle,
  SwitchRow,
  Text,
  TextField,
  showToast
} from "../../ui";
import { shareApproximateLocation } from "../../utils/location";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

export function SettingsScreen({ navigation }: Props) {
  const { profile, user, signOut, forgetSession } = useAuth();
  const [busy, setBusy] = useState(false);
  const [confirmLogout, setConfirmLogout] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);

  if (!profile) return null;

  const toggle = async (input: { discoverable?: boolean; showDistance?: boolean }) => {
    setBusy(true);
    try {
      setProfileCache(await updatePrivacy(input));
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const refreshLocation = async () => {
    setBusy(true);
    const outcome = await shareApproximateLocation();
    setBusy(false);
    if (outcome === "shared") {
      setProfileCache(await getMyProfile());
      showToast("Localisation approximative mise à jour.", "success");
    } else {
      showToast(outcome === "denied" ? "Autorisation de localisation refusée." : "Position indisponible.", "error");
    }
  };

  const removeLocation = async () => {
    setBusy(true);
    try {
      await clearLocation();
      setProfileCache(await getMyProfile());
      showToast("Localisation supprimée.", "success");
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const confirmDeletion = async () => {
    setBusy(true);
    setDeleteError(null);
    try {
      await deleteAccount(password);
      setDeleting(false);
      await forgetSession();
      showToast("Votre compte et vos données ont été supprimés.", "success");
    } catch (e) {
      setDeleteError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="settings-screen">
      <Header onBack={navigation.goBack} title="Confidentialité et sécurité" />

      <SectionTitle>Visibilité</SectionTitle>
      <Card>
        <SwitchRow
          detail="Désactivez pour mettre votre profil en pause. Vos matchs et conversations restent accessibles."
          disabled={busy}
          label="Apparaître dans la découverte"
          onChange={(v) => void toggle({ discoverable: v })}
          testID="settings-discoverable"
          value={profile.discoverable}
        />
        <SwitchRow
          detail="La distance affichée est toujours approximative."
          disabled={busy}
          label="Afficher ma distance"
          onChange={(v) => void toggle({ showDistance: v })}
          value={profile.showDistance}
        />
      </Card>

      <SectionTitle>Localisation</SectionTitle>
      <Card>
        <ListRow
          detail={profile.hasLocation ? "Position approximative (≈ 1 km) enregistrée" : "Désactivée : la distance n'est pas utilisée"}
          icon="navigate-outline"
          label={profile.hasLocation ? "Mettre à jour ma position" : "Activer la localisation"}
          onPress={refreshLocation}
        />
        {profile.hasLocation ? <ListRow icon="close-circle-outline" label="Supprimer ma position" onPress={removeLocation} /> : null}
      </Card>

      <SectionTitle>Sécurité</SectionTitle>
      <Card>
        <ListRow icon="key-outline" label="Changer mon mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
        <ListRow
          icon="hand-left-outline"
          label="Personnes bloquées"
          onPress={() => navigation.navigate("Blocked")}
          testID="settings-blocked"
        />
      </Card>

      <SectionTitle>Compte</SectionTitle>
      <Card>
        <ListRow
          detail={user?.email}
          icon="log-out-outline"
          label="Se déconnecter"
          onPress={() => setConfirmLogout(true)}
          testID="settings-logout"
        />
        <ListRow destructive icon="trash-outline" label="Supprimer mon compte" onPress={() => setDeleting(true)} testID="settings-delete" />
      </Card>
      <Text style={styles.note} tone="subtle" variant="caption">
        La suppression efface définitivement votre profil, vos photos, vos matchs, vos messages et vos notifications.
      </Text>

      <ConfirmDialog
        confirmLabel="Se déconnecter"
        message="Vous pourrez vous reconnecter à tout moment."
        onCancel={() => setConfirmLogout(false)}
        onConfirm={() => void signOut()}
        title="Se déconnecter ?"
        visible={confirmLogout}
      />
      <Dialog
        message="Cette action est irréversible. Confirmez avec votre mot de passe."
        onClose={() => setDeleting(false)}
        testID="delete-dialog"
        title="Supprimer mon compte"
        visible={deleting}
      >
        <TextField
          error={deleteError}
          label="Mot de passe"
          onChangeText={setPassword}
          secureToggle
          testID="delete-password"
          value={password}
        />
        <View style={styles.actions}>
          <Button
            disabled={!password}
            label="Supprimer définitivement"
            loading={busy}
            onPress={confirmDeletion}
            testID="delete-confirm"
            variant="danger"
          />
          <Button label="Annuler" onPress={() => setDeleting(false)} variant="ghost" />
        </View>
      </Dialog>
    </Screen>
  );
}

const styles = StyleSheet.create({
  note: { marginTop: spacing.sm, marginHorizontal: spacing.xxs },
  actions: { gap: spacing.xs }
});
