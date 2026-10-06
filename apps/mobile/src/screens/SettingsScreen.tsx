import React, { useState } from "react";
import { Alert, Modal, ScrollView, StyleSheet, Switch, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { savePrivacy } from "../api/dating";
import { deleteAccount } from "../api/platform";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Section } from "../shared/layout";
import { Button, Input, ListItem, Notice, Text, radii, spacing, useTheme } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

type PrivacyKey = "showAge" | "showDistance" | "discoverable";

export default function SettingsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { self, user, refreshSelf, signOut, endSession } = useAuth();
  const profile = self?.profile;
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");

  const privacy = useMutation({
    mutationFn: savePrivacy,
    onSuccess: refreshSelf,
    onError: (err) => showToast(describeError(err), "danger")
  });

  const toggle = (key: PrivacyKey, value: boolean) => {
    if (!profile) {
      return;
    }
    privacy.mutate({ showAge: profile.showAge, showDistance: profile.showDistance, discoverable: profile.discoverable, [key]: value });
  };

  const erase = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      setDeleting(false);
      await endSession();
      showToast("Votre compte et vos données ont été supprimés.", "success");
    }
  });

  const confirmSignOut = () =>
    Alert.alert("Se déconnecter ?", undefined, [
      { text: "Annuler", style: "cancel" },
      { text: "Se déconnecter", onPress: () => void signOut() }
    ]);

  const row = (key: PrivacyKey, title: string, subtitle: string) => (
    <View style={[styles.switchRow, { borderBottomColor: colors.border }]}>
      <View style={styles.switchCopy}>
        <Text weight="medium">{title}</Text>
        <Text tone="muted" variant="caption">
          {subtitle}
        </Text>
      </View>
      <Switch
        accessibilityLabel={title}
        disabled={!profile || privacy.isPending}
        onValueChange={(value) => toggle(key, value)}
        thumbColor="#FFFFFF"
        trackColor={{ true: colors.primary, false: colors.borderStrong }}
        value={profile ? profile[key] : false}
      />
    </View>
  );

  return (
    <ScrollView contentContainerStyle={styles.content} testID="settings-screen">
      <Section title="Confidentialité">
        {row(
          "discoverable",
          "Apparaître dans la découverte",
          "Désactivé : vos matchs gardent l'accès à votre profil, mais personne de nouveau ne vous voit."
        )}
        {row("showAge", "Afficher mon âge", "Sinon, seul votre prénom est visible.")}
        {row(
          "showDistance",
          "Afficher ma distance et ma ville",
          "La distance est arrondie par tranches de 5 km. Votre position exacte n'est jamais partagée."
        )}
      </Section>

      <Section title="Sécurité">
        <ListItem
          onPress={() => navigation.navigate("BlockedUsers")}
          subtitle="Gérer les personnes que vous avez bloquées"
          title="Personnes bloquées"
        />
      </Section>

      <Section title="Compte">
        <ListItem subtitle={user?.email} title="Adresse e-mail" />
        <ListItem onPress={confirmSignOut} testID="settings-logout" title="Se déconnecter" />
        <ListItem
          destructive
          onPress={() => setDeleting(true)}
          subtitle="Supprime définitivement votre profil, vos photos, vos matchs et vos messages."
          testID="settings-delete"
          title="Supprimer mon compte"
        />
      </Section>

      <Section title="À propos">
        <Text tone="muted" variant="caption">
          Lien est entièrement gratuit : pas d'abonnement, pas de boost payant, pas de limite de likes, pas de publicité.
        </Text>
      </Section>

      <Modal animationType="slide" onRequestClose={() => setDeleting(false)} transparent visible={deleting}>
        <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
          <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
            <Text variant="heading">Supprimer mon compte</Text>
            <Text tone="muted">
              Cette action est définitive : votre profil, vos photos, vos matchs et vos conversations seront effacés pour toujours.
              Saisissez votre mot de passe pour confirmer.
            </Text>
            <Input
              autoCapitalize="none"
              label="Mot de passe"
              onChangeText={setPassword}
              secureTextEntry
              testID="delete-password-input"
              value={password}
            />
            {erase.isError ? <Notice message={describeError(erase.error)} tone="danger" /> : null}
            <Button
              disabled={password.length === 0}
              label="Supprimer définitivement"
              loading={erase.isPending}
              onPress={() => erase.mutate()}
              testID="delete-confirm"
              variant="destructive"
            />
            <Button
              label="Annuler"
              onPress={() => {
                setDeleting(false);
                setPassword("");
                erase.reset();
              }}
              variant="ghost"
            />
          </View>
        </View>
      </Modal>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.xl },
  switchRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  switchCopy: { flex: 1, gap: 2 },
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.lg,
    paddingBottom: spacing.xxxl
  }
});
