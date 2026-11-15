import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../lib/errors";
import { genderLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Chip, ChipGroup, ErrorState, Header, IconButton, ListRow, Loading, PhotoImage, Screen, Section, Text } from "../../ui";
import { useMyProfile } from "./hooks";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function MyProfileScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const me = useMyProfile();

  if (me.isPending) return <Loading />;
  if (me.isError) return <ErrorState message={errorMessage(me.error)} onRetry={() => void me.refetch()} />;
  const { profile, preferences } = me.data;

  return (
    <Screen scroll testID="me-screen" refreshing={me.isRefetching} onRefresh={() => void me.refetch()}>
      <Header
        title="Mon profil"
        large
        right={
          <IconButton icon="settings-outline" label="Réglages" onPress={() => navigation.navigate("Settings")} testID="open-settings" />
        }
      />
      <View style={[styles.hero, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        <PhotoImage photo={profile.photos[0]} style={styles.heroPhoto} accessibilityLabel="Votre photo principale" />
        <View style={styles.heroText}>
          <Text variant="title">
            {profile.firstName}
            {profile.age ? `, ${profile.age}` : ""}
          </Text>
          <Text color="textMuted">
            {[profile.gender ? genderLabels[profile.gender] : null, profile.city || null].filter(Boolean).join(" · ")}
          </Text>
          {profile.isPaused ? (
            <Text variant="micro" color="accent">
              PROFIL EN PAUSE
            </Text>
          ) : null}
        </View>
      </View>

      <Section title="Profil">
        <ListRow
          icon="create-outline"
          label="Modifier mon profil"
          description="Photos, bio, centres d'intérêt"
          onPress={() => navigation.navigate("EditProfile")}
          testID="open-edit-profile"
        />
        <ListRow
          icon="heart-circle-outline"
          label="Préférences de rencontre"
          description={`${preferences.minAge}–${preferences.maxAge} ans · ${preferences.maxDistanceKm} km`}
          onPress={() => navigation.navigate("Preferences")}
        />
      </Section>

      {profile.bio ? (
        <Section title="Bio">
          <Text style={styles.padded}>{profile.bio}</Text>
        </Section>
      ) : null}

      {profile.interests.length ? (
        <Section title="Centres d'intérêt">
          <View style={styles.padded}>
            <ChipGroup>
              {profile.interests.map((i) => (
                <Chip key={i.id} label={i.label} />
              ))}
            </ChipGroup>
          </View>
        </Section>
      ) : null}

      <View style={[styles.free, { backgroundColor: colors.primarySoft }]}>
        <Text variant="label" color="primary">
          Lueur est 100 % gratuite
        </Text>
        <Text variant="caption" color="textMuted">
          Pas d’abonnement, pas de « boost », pas de likes limités. Toutes les fonctionnalités sont ouvertes à tous.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flexDirection: "row",
    gap: spacing.md,
    alignItems: "center",
    padding: spacing.md,
    borderRadius: radii.lg,
    borderWidth: 1,
    marginBottom: spacing.lg
  },
  heroPhoto: { width: 88, height: 112, borderRadius: radii.md },
  heroText: { flex: 1, gap: 4 },
  padded: { paddingVertical: spacing.sm },
  free: { padding: spacing.md, borderRadius: radii.lg, gap: 4 }
});
