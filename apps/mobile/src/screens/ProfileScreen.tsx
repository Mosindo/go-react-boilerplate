import React, { useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { saveProfile } from "../api/profile";
import type { SelfProfile } from "../api/types";
import { ConfirmDialog } from "../components/Dialogs";
import { PersonAvatar } from "../components/PersonAvatar";
import { useAuth } from "../hooks/useAuth";
import { useProfileQuery } from "../hooks/useProfileData";
import { genderLabel } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Card, Input, ListItem, Text, spacing } from "../shared/ui";

function toInput(profile: SelfProfile, patch: Partial<Pick<SelfProfile, "isDiscoverable" | "showDistance">>) {
  return {
    firstName: profile.firstName,
    birthdate: profile.birthdate,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    latitude: profile.latitude,
    longitude: profile.longitude,
    interestIds: profile.interests.map((interest) => interest.id),
    isDiscoverable: profile.isDiscoverable,
    showDistance: profile.showDistance,
    ...patch
  };
}

/** "Profil" tab: my card, privacy switches, safety lists, sign out and account deletion. */
export default function ProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const { signOut, deleteAccount, user } = useAuth();
  const query = useProfileQuery();
  const profile = query.data;

  const [saving, setSaving] = useState<string | null>(null);
  const [privacyError, setPrivacyError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  const setPrivacy = async (key: "isDiscoverable" | "showDistance", value: boolean) => {
    if (!profile) {
      return;
    }
    setSaving(key);
    setPrivacyError(null);
    try {
      await saveProfile(toInput(profile, { [key]: value }));
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
    } catch (err) {
      setPrivacyError(errorMessage(err));
    } finally {
      setSaving(null);
    }
  };

  const confirmDelete = async () => {
    setDeleteBusy(true);
    setDeleteError(null);
    try {
      await deleteAccount(password);
    } catch (err) {
      setDeleteError(errorMessage(err, "Suppression impossible."));
      setDeleteBusy(false);
    }
  };

  if (query.isLoading) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <LoadingView label="Chargement…" />
      </ScreenContainer>
    );
  }
  if (query.isError || !profile) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <ErrorView
          actionLabel="Réessayer"
          message={query.error ? errorMessage(query.error) : "Profil introuvable."}
          onAction={() => void query.refetch()}
        />
        <Button label="Se déconnecter" onPress={signOut} variant="ghost" />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer contentMaxWidth={560} edges={["top", "left", "right"]} testID="profile-screen">
      <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
        <View style={styles.hero}>
          <PersonAvatar name={profile.firstName} photo={profile.photos[0]} size={96} />
          <View style={styles.heroCopy}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              {profile.firstName}, {profile.age}
            </Text>
            <Text tone="muted">
              {genderLabel(profile.gender)} · {profile.city}
            </Text>
            <Text tone="muted" variant="caption">
              {user?.email}
            </Text>
          </View>
        </View>
        <Button label="Modifier mon profil" onPress={() => navigation.navigate("EditProfile")} testID="edit-profile" variant="secondary" />

        <Card padding="md" style={styles.card}>
          <Text variant="heading" weight="bold">
            Confidentialité
          </Text>
          <ListItem
            disabled={saving !== null}
            onPress={() => setPrivacy("isDiscoverable", !profile.isDiscoverable)}
            subtitle={profile.isDiscoverable ? "Visible par les autres" : "Mode invisible actif"}
            testID="toggle-discoverable"
            title="Apparaître dans la découverte"
            trailing={
              <Text weight="bold" tone={profile.isDiscoverable ? "success" : "muted"}>
                {profile.isDiscoverable ? "Oui" : "Non"}
              </Text>
            }
            variant="muted"
          />
          <ListItem
            disabled={saving !== null}
            onPress={() => setPrivacy("showDistance", !profile.showDistance)}
            subtitle="Approximative, jamais exacte"
            testID="toggle-distance"
            title="Afficher ma distance"
            trailing={
              <Text weight="bold" tone={profile.showDistance ? "success" : "muted"}>
                {profile.showDistance ? "Oui" : "Non"}
              </Text>
            }
            variant="muted"
          />
          {privacyError ? (
            <Text tone="danger" variant="label">
              {privacyError}
            </Text>
          ) : null}
        </Card>

        <Card padding="md" style={styles.card}>
          <Text variant="heading" weight="bold">
            Réglages
          </Text>
          <ListItem
            onPress={() => navigation.navigate("Preferences")}
            subtitle="Genre, âge, distance"
            testID="open-preferences-settings"
            title="Préférences de rencontre"
            variant="muted"
          />
          <ListItem onPress={() => navigation.navigate("BlockedUsers")} testID="open-blocked" title="Comptes bloqués" variant="muted" />
        </Card>

        <View style={styles.footer}>
          <Button label="Se déconnecter" onPress={signOut} testID="sign-out" variant="outline" />
          <Button
            label="Supprimer mon compte"
            onPress={() => setDeleting(true)}
            testID="delete-account"
            variant="ghost"
            textStyle={styles.danger}
          />
          <Text style={styles.free} tone="muted" variant="caption">
            Aurore est entièrement gratuite : aucun abonnement, aucune limite de likes, aucune publicité.
          </Text>
        </View>
      </ScrollView>

      <ConfirmDialog
        confirmLabel="Supprimer définitivement"
        destructive
        error={deleteError}
        loading={deleteBusy}
        message="Votre profil, vos photos, vos matchs et vos messages seront supprimés immédiatement et définitivement. Entrez votre mot de passe pour confirmer."
        onCancel={() => {
          setDeleting(false);
          setPassword("");
          setDeleteError(null);
        }}
        onConfirm={confirmDelete}
        title="Supprimer mon compte ?"
        visible={deleting}
      >
        <Input
          autoCapitalize="none"
          label="Mot de passe"
          onChangeText={setPassword}
          secureTextEntry
          testID="delete-password"
          value={password}
        />
      </ConfirmDialog>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.lg, paddingBottom: spacing.xxl },
  hero: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  heroCopy: { flex: 1, gap: spacing.xxs },
  card: { gap: spacing.sm },
  footer: { gap: spacing.sm, marginTop: spacing.md },
  danger: { color: "#b3261e" },
  free: { textAlign: "center", marginTop: spacing.sm }
});
