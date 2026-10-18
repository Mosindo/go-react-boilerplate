import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { PublicProfile } from "../api/types";
import { Text, colors, radii, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { PhotoImage } from "./PhotoImage";
import { formatDistance } from "./labels";

type ProfileCardProps = {
  profile: PublicProfile;
  /** "card" is the compact deck card; "full" shows bio and every interest. */
  variant?: "card" | "full";
  onOpenDetails?: () => void;
  testID?: string;
};

export function ProfileCard({ profile, variant = "card", onOpenDetails, testID }: ProfileCardProps) {
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];
  const distance = formatDistance(profile.distanceKm);
  const showAll = variant === "full";
  const interests = showAll ? profile.interests : profile.interests.slice(0, 3);

  const step = (delta: number) => setIndex((i) => Math.min(photos.length - 1, Math.max(0, i + delta)));

  return (
    <View style={[styles.card, showAll ? styles.full : null]} testID={testID}>
      <PhotoImage fallbackLabel={profile.firstName} path={current?.url} style={StyleSheet.absoluteFill} />

      {photos.length > 1 ? (
        <>
          <View pointerEvents="none" style={styles.dots}>
            {photos.map((photo, i) => (
              <View key={photo.id} style={[styles.dot, i === index ? styles.dotActive : null]} />
            ))}
          </View>
          <View style={styles.tapZones}>
            <Pressable
              accessibilityLabel="Photo précédente"
              onPress={() => step(-1)}
              style={styles.tapZone}
              testID="photo-prev"
            />
            <Pressable
              accessibilityLabel="Photo suivante"
              onPress={() => step(1)}
              style={styles.tapZone}
              testID="photo-next"
            />
          </View>
        </>
      ) : null}

      <LinearGradient colors={["rgba(20,12,22,0)", "rgba(20,12,22,0.78)"]} pointerEvents="none" style={styles.scrim} />

      <View style={styles.info}>
        <Pressable
          accessibilityRole={onOpenDetails ? "button" : undefined}
          disabled={!onOpenDetails}
          onPress={onOpenDetails}
        >
          <Text style={styles.name} tone="inverse" weight="bold">
            {profile.firstName}, {profile.age}
          </Text>
          {profile.city || distance ? (
            <Text tone="inverse" variant="label">
              {[profile.city, distance].filter(Boolean).join(" · ")}
            </Text>
          ) : null}
        </Pressable>
        {showAll && profile.bio ? (
          <Text style={styles.bio} tone="inverse">
            {profile.bio}
          </Text>
        ) : null}
        {interests.length > 0 ? (
          <ChipRow>
            {interests.map((interest) => (
              <Chip key={interest.slug} label={interest.label} />
            ))}
          </ChipRow>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    flex: 1,
    borderRadius: radii.xl,
    overflow: "hidden",
    backgroundColor: colors.surfaceSubtle,
    justifyContent: "flex-end"
  },
  full: { minHeight: 520 },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.4)" },
  dotActive: { backgroundColor: "#ffffff" },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  tapZone: { flex: 1 },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0, height: 280 },
  info: { padding: spacing.lg, gap: spacing.sm },
  name: { fontSize: 28, lineHeight: 34 },
  bio: { marginTop: spacing.xs }
});
