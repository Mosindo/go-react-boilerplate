import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { PublicProfile } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { Chip, Text, radii, spacing } from "../../shared/ui";
import { PhotoStack } from "../profile/PhotoStack";

type ProfileCardProps = {
  profile: PublicProfile;
  onOpenDetails: () => void;
};

/** Full-bleed photo card with the essentials overlaid. */
export function ProfileCard({ profile, onOpenDetails }: ProfileCardProps) {
  const distance = formatDistance(profile.distanceKm);
  const place = [distance, profile.city].filter(Boolean).join(" · ");
  return (
    <PhotoStack name={profile.firstName} photos={profile.photos} style={styles.card}>
      <Pressable
        accessibilityHint="Affiche le profil complet"
        accessibilityLabel={`${profile.firstName}, ${profile.age} ans. Voir le profil`}
        accessibilityRole="button"
        onPress={onOpenDetails}
        style={styles.info}
        testID="profile-card-details"
      >
        <View style={styles.titleRow}>
          <Text style={styles.white} variant="heading" weight="bold">
            {profile.firstName}, {profile.age}
          </Text>
          <Ionicons color="#FFFFFF" name="information-circle-outline" size={24} />
        </View>
        {place ? (
          <View style={styles.row}>
            <Ionicons color="#FFFFFF" name="location-outline" size={14} />
            <Text style={styles.white} variant="label">
              {place}
            </Text>
          </View>
        ) : null}
        {profile.bio ? (
          <Text numberOfLines={2} style={styles.white} variant="label">
            {profile.bio}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 3).map((interest) => (
              <Chip key={interest.id} label={interest.label} />
            ))}
          </View>
        ) : null}
      </Pressable>
    </PhotoStack>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, gap: spacing.xs },
  titleRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xxs },
  chips: { flexDirection: "row", gap: spacing.xs, flexWrap: "wrap", paddingTop: spacing.xs },
  white: { color: "#FFFFFF" }
});
