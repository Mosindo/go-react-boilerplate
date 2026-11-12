import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../../hooks/useAuth";
import { useOwnProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { ListRow, Loading, Photo, Screen, Section, Text } from "../../ui";
import { nameAge } from "../../utils/format";

export default function MeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { colors } = useTheme();
  const { user, signOut } = useAuth();
  const { data: profile } = useOwnProfile();
  if (!profile) return <Loading />;

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Photo url={profile.photos[0]?.url} style={[styles.avatar, { borderColor: colors.primary }]} accessibilityLabel="Votre photo principale" />
        <Text variant="title" serif accessibilityRole="header">
          {nameAge(profile.firstName, profile.age)}
        </Text>
        <Text tone="muted">{user?.email}</Text>
        {!profile.privacy.discoverable ? (
          <View style={[styles.pill, { backgroundColor: colors.accentSoft }]}>
            <Text variant="caption" style={{ color: colors.accent }}>
              Profil en pause : vous n’apparaissez pas dans la découverte
            </Text>
          </View>
        ) : null}
      </View>

      <Section title="Mon profil">
        <ListRow
          icon="eye-outline"
          title="Aperçu de mon profil"
          subtitle="Tel que les autres le voient"
          onPress={() => navigation.navigate("ProfileView", { userId: profile.userId })}
        />
        <ListRow icon="person-outline" title="Informations" subtitle="Prénom, bio, métier, ville" onPress={() => navigation.navigate("EditProfile")} />
        <ListRow icon="images-outline" title="Photos" subtitle={`${profile.photos.length}/6`} onPress={() => navigation.navigate("EditPhotos")} />
        <ListRow
          icon="sparkles-outline"
          title="Centres d'intérêt"
          subtitle={`${profile.interests.length} sélectionnés`}
          onPress={() => navigation.navigate("EditInterests")}
          last
        />
      </Section>

      <Section title="Rencontres">
        <ListRow
          icon="options-outline"
          title="Préférences"
          subtitle={`${profile.preferences.minAge}–${profile.preferences.maxAge} ans · ${profile.preferences.maxDistanceKm} km`}
          onPress={() => navigation.navigate("EditPreferences")}
        />
        <ListRow icon="shield-checkmark-outline" title="Confidentialité et localisation" onPress={() => navigation.navigate("Privacy")} />
        <ListRow icon="ban-outline" title="Personnes bloquées" onPress={() => navigation.navigate("Blocked")} last />
      </Section>

      <Section title="Compte" footer="Lueur est entièrement gratuit : pas d'abonnement, pas de publicité, pas de limite de likes.">
        <ListRow icon="key-outline" title="Mot de passe et suppression" onPress={() => navigation.navigate("Account")} />
        <ListRow icon="log-out-outline" title="Se déconnecter" onPress={() => void signOut()} last />
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, paddingTop: spacing.lg },
  avatar: { width: 112, height: 112, borderRadius: 56, borderWidth: 3, marginBottom: spacing.xs },
  pill: { borderRadius: 999, paddingHorizontal: spacing.sm, paddingVertical: 4, marginTop: spacing.xs }
});
