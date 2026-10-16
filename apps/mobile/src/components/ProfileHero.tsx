import React from "react";
import { StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { formatDistance } from "../lib/format";
import { Text, colors, radii, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { PhotoCarousel } from "./PhotoCarousel";

type ProfileHeroProps = {
  profile: PublicProfile;
  onCenterPress?: () => void;
  /** Compact cards (swipe deck) show only a few interests. */
  maxInterests?: number;
};

/** Photo area with name/age/distance overlaid; the building block of the swipe card. */
export function ProfileHero({ profile, onCenterPress, maxInterests = 3 }: ProfileHeroProps) {
  const distance = formatDistance(profile.distanceKm);
  return (
    <View style={styles.root}>
      <PhotoCarousel name={profile.firstName} onCenterPress={onCenterPress} photos={profile.photos} />
      <View pointerEvents="none" style={styles.scrim}>
        <Text style={styles.name} tone="inverse" variant="title" weight="bold">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.occupation ? (
          <Text tone="inverse" weight="medium">
            {profile.occupation}
          </Text>
        ) : null}
        {distance || profile.city ? (
          <Text style={styles.sub} tone="inverse" variant="label">
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        ) : null}
        {maxInterests > 0 && profile.interests.length > 0 ? (
          <ChipRow>
            {profile.interests.slice(0, maxInterests).map((interest) => (
              <Chip key={interest.slug} label={interest.label} tone="highlight" />
            ))}
          </ChipRow>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  scrim: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxxl,
    gap: spacing.xs,
    backgroundColor: colors.scrim
  },
  name: { textShadowColor: "rgba(0,0,0,0.3)", textShadowRadius: 6 },
  sub: { opacity: 0.9 }
});
