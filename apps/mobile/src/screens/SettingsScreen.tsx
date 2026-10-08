import React, { useState } from "react";
import { Switch } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { deleteAccount } from "../api/auth";
import { describeError } from "../api/client";
import { useMe, useSavePreferences } from "../hooks/data";
import { useAuth } from "../hooks/useAuth";
import type { MainStackParamList } from "../navigation/types";
import { LoadingView, showToast } from "../shared/feedback";
import { ScreenContainer, Section } from "../shared/layout";
import { Button, Card, FormField, ListItem, Notice, Sheet, Text } from "../shared/ui";
import { useTheme } from "../theme";
import { confirm } from "../utils/confirm";

export default function SettingsScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Settings">) {
  const { colors } = useTheme();
  const { data: me } = useMe();
  const save = useSavePreferences();
  const { signOut, forgetSession } = useAuth();
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (!me) return <LoadingView fullScreen />;
  const prefs = me.preferences;

  const toggle = async (patch: Partial<typeof prefs>) => {
    try {
      await save.mutateAsync({ ...prefs, ...patch });
    } catch (error) {
      showToast(describeError(error), "error");
    }
  };

  const confirmDelete = async () => {
    if (!password) {
      setDeleteError("Saisissez votre mot de passe pour confirmer.");
      return;
    }
    if (
      !(await confirm({
        title: "Supprimer définitivement mon compte ?",
        message: "Profil, photos, matchs et messages seront effacés. Cette action est irréversible.",
        confirmLabel: "Supprimer",
        destructive: true
      }))
    )
      return;
    setBusy(true);
    setDeleteError(null);
    try {
      await deleteAccount(password);
      setDeleting(false);
      await forgetSession();
    } catch (error) {
      setDeleteError(describeError(error));
      setBusy(false);
    }
  };

  const switchColors = { trackColor: { false: colors.border, true: colors.primary }, thumbColor: "#FFFFFF" };

  return (
    <ScreenContainer>
      <Section title="Confidentialité">
        <Card>
          <ListItem
            chevron={false}
            icon="pause-circle-outline"
            right={
              <Switch
                accessibilityLabel="Mettre mon profil en pause"
                onValueChange={(isVisible) => void toggle({ isVisible: !isVisible })}
                value={!prefs.isVisible}
                {...switchColors}
              />
            }
            subtitle="Vous n'apparaissez plus dans la découverte. Vos matchs et conversations restent actifs."
            title="Mettre mon profil en pause"
          />
          <ListItem
            chevron={false}
            icon="navigate-outline"
            right={
              <Switch
                accessibilityLabel="Afficher ma distance"
                onValueChange={(showDistance) => void toggle({ showDistance })}
                value={prefs.showDistance}
                {...switchColors}
              />
            }
            subtitle="Les autres voient une distance approximative, jamais votre position."
            title="Afficher ma distance"
          />
          <ListItem icon="ban-outline" onPress={() => navigation.navigate("Blocked")} title="Personnes bloquées" />
        </Card>
      </Section>

      <Section title="Compte">
        <Card>
          <ListItem icon="mail-outline" subtitle={me.user.email} title="Adresse e-mail" />
          <ListItem icon="key-outline" onPress={() => navigation.navigate("ChangePassword")} title="Changer le mot de passe" />
          <ListItem icon="log-out-outline" onPress={() => void signOut()} title="Se déconnecter" />
          <ListItem destructive icon="trash-outline" onPress={() => setDeleting(true)} title="Supprimer mon compte" />
        </Card>
      </Section>

      <Text align="center" tone="subtle" variant="caption">
        Lien est gratuit et le restera : aucun abonnement, aucune fonctionnalité verrouillée, aucune publicité.
      </Text>

      <Sheet onClose={() => setDeleting(false)} title="Supprimer mon compte" visible={deleting}>
        <Text tone="muted">Toutes vos données seront définitivement supprimées. Saisissez votre mot de passe pour confirmer.</Text>
        {deleteError ? <Notice message={deleteError} tone="error" /> : null}
        <FormField autoCapitalize="none" label="Mot de passe" onChangeText={setPassword} secureTextEntry value={password} />
        <Button loading={busy} onPress={confirmDelete} variant="destructive">
          Supprimer définitivement
        </Button>
      </Sheet>
    </ScreenContainer>
  );
}
