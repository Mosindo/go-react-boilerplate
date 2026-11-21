import React, { useState } from "react";
import { Modal, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi, profileApi } from "../../api/endpoints";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles, useThemeContext, type ThemePreference } from "../../theme";
import { Button, Card, Chip, Header, ListRow, LoadingState, modalAnimation, Screen, Section, showToast, Text, TextField } from "../../ui";
import { useAuth } from "../auth/AuthProvider";
import { useMyProfile, useSetMyProfile } from "./hooks";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

const themeOptions: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "Système" },
  { value: "light", label: "Clair" },
  { value: "dark", label: "Sombre" }
];

export function SettingsScreen({ navigation }: Props) {
  const styles = useStyles();
  const me = useMyProfile().data;
  const setProfile = useSetMyProfile();
  const { preference, setPreference } = useThemeContext();
  const { forgetSession } = useAuth();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [password, setPassword] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  if (!me?.profile) return <LoadingState />;

  const toggle = async (key: "isVisible" | "showDistance", value: boolean) => {
    try {
      setProfile(await profileApi.settings({ [key]: value }));
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const deleteAccount = async () => {
    setDeleting(true);
    setDeleteError(null);
    try {
      await authApi.deleteAccount(password);
      setDeleteOpen(false);
      await forgetSession();
      showToast("Votre compte et vos données ont été supprimés.", "success");
    } catch (err) {
      setDeleteError(errorMessage(err));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <Screen scroll>
      <Header title="Confidentialité et sécurité" />
      <Section title="Visibilité">
        <Card>
          <ListRow
            testID="settings-visible"
            icon="eye-outline"
            label="Apparaître dans la découverte"
            description="Désactivez pour mettre votre profil en pause. Vos matchs restent actifs."
            value={me.profile.isVisible}
            onValueChange={(v) => void toggle("isVisible", v)}
          />
          <ListRow
            testID="settings-distance"
            icon="location-outline"
            label="Afficher ma distance"
            description="Les autres voient une distance arrondie, jamais votre position."
            value={me.profile.showDistance}
            onValueChange={(v) => void toggle("showDistance", v)}
            divider={false}
          />
        </Card>
      </Section>
      <Section title="Sécurité">
        <Card>
          <ListRow
            testID="settings-blocked"
            icon="ban-outline"
            label="Personnes bloquées"
            onPress={() => navigation.navigate("BlockedUsers")}
            divider={false}
          />
        </Card>
      </Section>
      <Section title="Apparence">
        <View style={styles.chips}>
          {themeOptions.map((o) => (
            <Chip key={o.value} label={o.label} selected={preference === o.value} onPress={() => setPreference(o.value)} />
          ))}
        </View>
      </Section>
      <Section title="Compte" hint={`Connecté avec ${me.email}`}>
        <Card>
          <ListRow
            testID="settings-delete"
            icon="trash-outline"
            label="Supprimer mon compte"
            description="Suppression définitive du profil, des photos, des matchs et des messages."
            tone="danger"
            onPress={() => setDeleteOpen(true)}
            divider={false}
          />
        </Card>
      </Section>

      <Modal visible={deleteOpen} transparent animationType={modalAnimation("fade")} onRequestClose={() => setDeleteOpen(false)}>
        <View style={styles.backdrop}>
          <View style={styles.dialog} accessibilityViewIsModal>
            <Text variant="title" accessibilityRole="header">
              Supprimer définitivement ?
            </Text>
            <Text tone="muted">
              Votre profil, vos photos, vos matchs et vos conversations seront effacés. Cette action est irréversible.
            </Text>
            <TextField
              testID="delete-password"
              label="Confirmez avec votre mot de passe"
              value={password}
              onChangeText={setPassword}
              secureToggle
              error={deleteError}
            />
            <Button
              testID="delete-confirm"
              label="Supprimer mon compte"
              variant="danger"
              size="lg"
              loading={deleting}
              disabled={!password}
              onPress={() => void deleteAccount()}
            />
            <Button label="Annuler" variant="ghost" onPress={() => setDeleteOpen(false)} />
          </View>
        </View>
      </Modal>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  chips: { flexDirection: "row", gap: t.spacing.sm },
  backdrop: { flex: 1, backgroundColor: t.colors.overlay, alignItems: "center", justifyContent: "center", padding: t.spacing.xl },
  dialog: {
    width: "100%",
    maxWidth: 440,
    backgroundColor: t.colors.surfaceRaised,
    borderRadius: t.radii.xl,
    padding: t.spacing.xl,
    gap: t.spacing.md
  }
}));
