import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage } from "../../api/client";
import type { PublicProfile } from "../../api/types";
import { SafetyActions } from "../../components/SafetyActions";
import { usePublicProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { colors, layout, radii, spacing } from "../../theme";
import { Chip, ErrorState, Header, IconButton, LoadingState, RemoteImage, Screen, Text } from "../../ui";
import { formatDistance } from "../../utils/format";
import { genderLabels, lookingForLabels } from "../../utils/labels";

export function ProfileView({ profile }: { profile: PublicProfile }) {
  const { width } = useWindowDimensions();
  const photoWidth = Math.min(width, layout.maxContentWidth) - spacing.lg * 2;
  const common = new Set(profile.commonInterestIds);
  const facts: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [];
  const distance = formatDistance(profile.distanceKm);
  if (distance) facts.push({ icon: "location-outline", text: distance });
  if (profile.city) facts.push({ icon: "business-outline", text: profile.city });
  if (profile.jobTitle) facts.push({ icon: "briefcase-outline", text: profile.jobTitle });
  if (profile.heightCm) facts.push({ icon: "resize-outline", text: `${profile.heightCm} cm` });
  facts.push({ icon: "person-outline", text: genderLabels[profile.gender] });
  if (profile.lookingFor) facts.push({ icon: "heart-outline", text: lookingForLabels[profile.lookingFor] });

  return (
    <View style={styles.wrap}>
      <View style={styles.titleRow}>
        <Text variant="display">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.isDemo ? (
          <View style={styles.demo}>
            <Text variant="caption" color="onPrimary">
              Démo
            </Text>
          </View>
        ) : null}
      </View>
      {profile.photos.slice(0, 1).map((p) => (
        <RemoteImage key={p.id} path={p.url} style={[styles.photo, { width: photoWidth, height: photoWidth * 1.25 }]} priority="high" />
      ))}
      {profile.bio ? (
        <View style={styles.section}>
          <Text variant="label" color="textSubtle">
            À PROPOS
          </Text>
          <Text variant="body">{profile.bio}</Text>
        </View>
      ) : null}
      <View style={styles.facts}>
        {facts.map((f) => (
          <View key={f.text} style={styles.fact}>
            <Ionicons name={f.icon} size={16} color={colors.gold} />
            <Text variant="small">{f.text}</Text>
          </View>
        ))}
      </View>
      {profile.interests.length ? (
        <View style={styles.section}>
          <Text variant="label" color="textSubtle">
            CENTRES D’INTÉRÊT
          </Text>
          <View style={styles.chips}>
            {profile.interests.map((i) => (
              <Chip key={i.id} label={i.label} highlighted={common.has(i.id)} />
            ))}
          </View>
          {common.size ? (
            <Text variant="caption" color="gold">
              {common.size} centre{common.size > 1 ? "s" : ""} d’intérêt en commun
            </Text>
          ) : null}
        </View>
      ) : null}
      {profile.photos.slice(1).map((p) => (
        <RemoteImage key={p.id} path={p.url} style={[styles.photo, { width: photoWidth, height: photoWidth * 1.25 }]} />
      ))}
    </View>
  );
}

export function ProfileDetailScreen({ navigation, route }: AppScreenProps<"ProfileDetail">) {
  const { userId, matchId } = route.params;
  const profile = usePublicProfile(userId);
  const [actionsOpen, setActionsOpen] = useState(false);

  return (
    <Screen edges={["top"]} testID="profile-detail-screen">
      <Header
        onBack={navigation.goBack}
        right={
          profile.data ? (
            <IconButton
              icon="ellipsis-horizontal"
              label="Signaler, bloquer…"
              onPress={() => setActionsOpen(true)}
              testID="profile-actions"
            />
          ) : null
        }
      />
      {profile.isLoading ? (
        <LoadingState />
      ) : profile.error || !profile.data ? (
        <ErrorState message={errorMessage(profile.error, "Profil introuvable.")} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={styles.scroll}>
          <ProfileView profile={profile.data} />
        </ScrollView>
      )}
      {profile.data ? (
        <SafetyActions
          visible={actionsOpen}
          onClose={() => setActionsOpen(false)}
          userId={userId}
          firstName={profile.data.firstName}
          matchId={matchId}
          onDone={() => navigation.navigate("Tabs")}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  scroll: {
    paddingBottom: spacing.xxl
  },
  wrap: {
    gap: spacing.lg
  },
  titleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  demo: {
    backgroundColor: colors.gold,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  },
  photo: {
    borderRadius: radii.lg
  },
  section: {
    gap: spacing.xs
  },
  facts: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  fact: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xxs,
    backgroundColor: colors.surface,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xxs
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
