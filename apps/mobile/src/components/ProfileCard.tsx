import React from "react";
import { StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { Chip, Text, radii, spacing } from "../shared/ui";
import { formatDistance } from "../utils/format";
import { PhotoCarousel } from "./PhotoCarousel";

type ProfileCardProps = { profile: PublicProfile };

/** The swipeable card: photo carousel with the essentials over a soft scrim. */
export function ProfileCard({ profile }: ProfileCardProps) {
  const distance = formatDistance(profile.distanceKm);
  const place = [profile.city, distance].filter(Boolean).join(" · ");
  return (
    <View style={styles.card}>
      <PhotoCarousel name={profile.firstName} photos={profile.photos} style={StyleSheet.absoluteFill} />
      <View pointerEvents="none" style={styles.scrim}>
        <Text accessibilityRole="header" style={styles.white} variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {place ? (
          <Text style={styles.whiteSoft} variant="label" weight="semibold">
            {place}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 3).map((interest) => (
              <Chip key={interest.slug} label={interest.label} selected />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: "#000" },
  scrim: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    gap: spacing.xs,
    backgroundColor: "rgba(20, 12, 18, 0.5)"
  },
  white: { color: "#ffffff" },
  whiteSoft: { color: "rgba(255,255,255,0.88)" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: spacing.xs }
});
