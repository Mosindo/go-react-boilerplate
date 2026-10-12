import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { deleteAccount } from "../api/auth";
import { errorMessage } from "../api/client";
import type { Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesFields } from "../components/PreferencesForm";
import { keys, useMyProfile, useSavePreferences, useSavePrivacy } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { useLocationUpdate } from "../hooks/useLocationUpdate";
import type { MainStackParamList } from "../navigation/types";
import { radii, spacing, useTheme, type ThemeMode } from "../theme";
import { Button, Chip, ErrorState, Input, LoadingState, Screen, Text, useToast } from "../ui";

const THEME_LABELS: Record<ThemeMode, string> = { system: "Automatique", light: "Clair", dark: "Sombre" };

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.section, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <Text variant="heading">{title}</Text>
      {children}
    </View>
  );
}

export default function SettingsScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Settings">) {
  const { user, signOut, forgetSession } = useAuth();
  const { mode, setMode } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const query = useMyProfile();
  const savePrefs = useSavePreferences();
  const savePrivacy = useSavePrivacy();
  const location = useLocationUpdate();

  const [prefs, setPrefs] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [password, setPassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (query.data?.preferences) {
      setPrefs(query.data.preferences);
    }
  }, [query.data?.preferences]);

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError || !query.data?.profile) {
    return <ErrorState message={errorMessage(query.error, "Réglages indisponibles.")} onRetry={() => void query.refetch()} />;
  }
  const profile = query.data.profile;

  const toggle = async (input: { isVisible?: boolean; showDistance?: boolean }) => {
    try {
      await savePrivacy.mutateAsync(input);
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  const savePreferences = async () => {
    if (prefs.interestedIn.length === 0) {
      toast.show("Choisissez au moins une option.", "error");
      return;
    }
    try {
      await savePrefs.mutateAsync(prefs);
      toast.show("Préférences enregistrées", "success");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  const refreshLocation = async () => {
    const result = await location.update();
    if (result.ok) {
      client.setQueryData(keys.me, result.profile);
      void client.invalidateQueries({ queryKey: keys.discover });
      toast.show("Position actualisée (approximative)", "success");
    } else {
      toast.show(
        result.reason === "denied" ? "Autorisez la localisation dans les réglages du téléphone." : "Position indisponible.",
        "error"
      );
    }
  };

  const remove = async () => {
    if (!password) {
      toast.show("Saisissez votre mot de passe pour confirmer.", "error");
      return;
    }
    setDeleting(true);
    try {
      await deleteAccount(password);
      await forgetSession();
    } catch (e) {
      toast.show(errorMessage(e), "error");
      setDeleting(false);
    }
  };

  return (
    <Screen scroll top={false}>
      <Section title="Préférences de rencontre">
        <PreferencesFields value={prefs} onChange={setPrefs} />
        <Button label="Enregistrer" variant="secondary" onPress={() => void savePreferences()} loading={savePrefs.isPending} />
      </Section>

      <Section title="Confidentialité">
        <View style={styles.chips}>
          <Chip
            label={profile.isVisible ? "Profil visible" : "Profil masqué"}
            selected={profile.isVisible}
            onPress={() => void toggle({ isVisible: !profile.isVisible })}
          />
          <Chip
            label={profile.showDistance ? "Distance affichée" : "Distance masquée"}
            selected={profile.showDistance}
            onPress={() => void toggle({ showDistance: !profile.showDistance })}
          />
        </View>
        <Text variant="caption" tone="muted">
          Masqué, votre profil n'apparaît plus dans la découverte. Votre position est toujours arrondie à ~1 km et n'est jamais affichée
          précisément.
        </Text>
        <Button
          label={profile.hasLocation ? "Actualiser ma position" : "Activer la localisation"}
          variant="secondary"
          onPress={() => void refreshLocation()}
          loading={location.loading}
        />
        <Button label="Personnes bloquées" variant="secondary" onPress={() => navigation.navigate("Blocked")} />
      </Section>

      <Section title="Apparence">
        <View style={styles.chips}>
          {(Object.keys(THEME_LABELS) as ThemeMode[]).map((m) => (
            <Chip key={m} label={THEME_LABELS[m]} selected={mode === m} onPress={() => setMode(m)} />
          ))}
        </View>
      </Section>

      <Section title="Compte">
        <Text tone="muted">{user?.email}</Text>
        <Button label="Se déconnecter" variant="secondary" onPress={() => void signOut()} />
        {confirmingDelete ? (
          <View style={styles.danger}>
            <Text tone="danger">Cette action est définitive : votre profil, vos photos, vos matchs et vos messages seront supprimés.</Text>
            <Input label="Mot de passe" value={password} onChangeText={setPassword} secureTextEntry autoCapitalize="none" />
            <Button label="Supprimer définitivement mon compte" variant="danger" onPress={() => void remove()} loading={deleting} />
            <Button label="Annuler" variant="ghost" onPress={() => setConfirmingDelete(false)} />
          </View>
        ) : (
          <Button label="Supprimer mon compte" variant="ghost" onPress={() => setConfirmingDelete(true)} />
        )}
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  section: { borderWidth: 1, borderRadius: radii.lg, padding: spacing.lg, gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  danger: { gap: spacing.md }
});
