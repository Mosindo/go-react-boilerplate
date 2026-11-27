import React from "react";
import { StyleSheet, View } from "react-native";
import { APP_VERSION } from "../../config";
import { useMyProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { useTheme, type ThemePreference } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/palette";
import { Card, Chip, Header, ListRow, RemoteImage, Screen, Section, Text } from "../../ui";
import type { TabScreenProps } from "../../navigation/types";

const THEMES: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "Système" },
  { value: "light", label: "Clair" },
  { value: "dark", label: "Sombre" }
];

export function MeScreen({ navigation }: TabScreenProps<"Me">) {
  const { signOut, user } = useAuth();
  const { preference, setPreference } = useTheme();
  const profile = useMyProfile().data;

  return (
    <Screen scroll testID="me-screen">
      <Header title="Moi" large />
      <View style={styles.content}>
        <Card style={styles.hero}>
          <RemoteImage url={profile?.photos[0]?.url} style={styles.avatar} accessibilityLabel="Ma photo principale" />
          <View style={styles.flex}>
            <Text variant="title">
              {profile?.firstName}
              {profile?.age ? `, ${profile.age}` : ""}
            </Text>
            <Text variant="caption" tone="muted">
              {user?.email}
            </Text>
            {profile && !profile.isVisible ? (
              <Text variant="caption" tone="accent">
                Profil en pause : tu n’apparais pas dans Découvrir.
              </Text>
            ) : null}
          </View>
        </Card>

        <Section title="Mon profil">
          <Card>
            <ListRow icon="create-outline" title="Modifier mon profil" subtitle="Photos, bio, centres d'intérêt" onPress={() => navigation.navigate("EditProfile")} testID="me-edit" />
            <ListRow icon="options-outline" title="Critères de recherche" subtitle="Genres, âge, distance" onPress={() => navigation.navigate("Preferences")} testID="me-preferences" />
          </Card>
        </Section>

        <Section title="Confidentialité et sécurité">
          <Card>
            <ListRow icon="eye-off-outline" title="Confidentialité" subtitle="Visibilité, distance, position" onPress={() => navigation.navigate("Privacy")} testID="me-privacy" />
            <ListRow icon="ban-outline" title="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} />
          </Card>
        </Section>

        <Section title="Apparence">
          <View style={styles.chips}>
            {THEMES.map((t) => (
              <Chip key={t.value} label={t.label} selected={preference === t.value} onPress={() => setPreference(t.value)} />
            ))}
          </View>
        </Section>

        <Section title="Compte">
          <Card>
            <ListRow icon="log-out-outline" title="Se déconnecter" onPress={() => void signOut()} testID="me-logout" />
            <ListRow icon="trash-outline" title="Supprimer mon compte" destructive onPress={() => navigation.navigate("DeleteAccount")} testID="me-delete" />
          </Card>
        </Section>

        <Text variant="caption" tone="subtle" align="center" style={styles.footer}>
          Lueur {APP_VERSION} · 100 % gratuit, sans abonnement ni publicité.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center" },
  hero: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  avatar: { width: 84, height: 112, borderRadius: radii.md },
  flex: { flex: 1, gap: 2 },
  chips: { flexDirection: "row", gap: spacing.xs },
  footer: { marginTop: spacing.xl }
});
