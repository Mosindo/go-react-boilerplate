import React from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { PublicProfile } from "../../api/types";
import { GENDER_LABELS, formatDistance } from "../../lib/format";
import { Chip, Text, radii, spacing } from "../../shared/ui";
import { useTheme } from "../../theme";
import { PhotoStack } from "./PhotoStack";

/** The full profile as other members see it. */
export function ProfileDetailView({ profile }: { profile: PublicProfile }) {
  const { colors } = useTheme();
  const distance = formatDistance(profile.distanceKm);
  return (
    <ScrollView contentContainerStyle={styles.scroll} testID="profile-detail">
      <PhotoStack name={profile.firstName} photos={profile.photos} style={styles.photo} />
      <View style={styles.body}>
        <View style={styles.titleRow}>
          <Text accessibilityRole="header" variant="title" weight="bold">
            {profile.firstName}, {profile.age}
          </Text>
        </View>
        <View style={styles.meta}>
          <View style={styles.row}>
            <Ionicons color={colors.textMuted} name="person-outline" size={16} />
            <Text tone="muted">{GENDER_LABELS[profile.gender]}</Text>
          </View>
          {distance || profile.city ? (
            <View style={styles.row}>
              <Ionicons color={colors.textMuted} name="location-outline" size={16} />
              <Text tone="muted">{[distance, profile.city].filter(Boolean).join(" · ")}</Text>
            </View>
          ) : null}
        </View>
        {profile.bio ? (
          <View style={styles.block}>
            <Text tone="muted" variant="eyebrow" weight="bold">
              À propos
            </Text>
            <Text>{profile.bio}</Text>
          </View>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.block}>
            <Text tone="muted" variant="eyebrow" weight="bold">
              Centres d'intérêt
            </Text>
            <View style={styles.chips}>
              {profile.interests.map((interest) => (
                <Chip key={interest.id} label={interest.label} />
              ))}
            </View>
          </View>
        ) : null}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scroll: { paddingBottom: 140 },
  photo: { width: "100%", aspectRatio: 3 / 4, maxHeight: 560, borderBottomLeftRadius: radii.xl, borderBottomRightRadius: radii.xl },
  body: { padding: spacing.lg, gap: spacing.lg, width: "100%", maxWidth: 560, alignSelf: "center" },
  titleRow: { flexDirection: "row", alignItems: "center" },
  meta: { gap: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  block: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
