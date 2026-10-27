import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { describeDistance } from "../lib/format";
import { radii, usePalette } from "../theme";
import { AuthImage } from "../ui/AuthImage";
import { Text } from "../ui/Text";

type Props = { profile: PublicProfile; onInfo?: () => void };

/** A discovery card: swipe through photos by tapping the left/right half. */
export function ProfileCard({ profile, onInfo }: Props) {
  const p = usePalette();
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const current = photos[Math.min(index, photos.length - 1)];
  const distance = describeDistance(profile.distanceKm);

  const go = (delta: number) =>
    setIndex((i) => Math.max(0, Math.min(photos.length - 1, i + delta)));

  return (
    <View
      style={[styles.card, { backgroundColor: p.surfaceAlt }]}
      testID={`card-${profile.userId}`}
    >
      <AuthImage
        path={current?.url}
        label={`Photo of ${profile.firstName}`}
        style={StyleSheet.absoluteFill}
      />
      <View style={styles.tapRow}>
        <Pressable
          style={styles.tap}
          onPress={() => go(-1)}
          accessibilityRole="button"
          accessibilityLabel="Previous photo"
        />
        <Pressable
          style={styles.tap}
          onPress={() => go(1)}
          accessibilityRole="button"
          accessibilityLabel="Next photo"
        />
      </View>
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <View style={[styles.caption, { backgroundColor: p.scrim }]} pointerEvents="box-none">
        <View style={styles.titleRow}>
          <Text variant="title" tone="inverse" style={styles.flex} numberOfLines={1}>
            {profile.firstName}, {profile.age}
          </Text>
          {onInfo ? (
            <Pressable
              testID={`info-${profile.userId}`}
              onPress={onInfo}
              accessibilityRole="button"
              accessibilityLabel={`See ${profile.firstName}'s full profile`}
              style={styles.info}
              hitSlop={10}
            >
              <Text variant="label" tone="inverse">
                i
              </Text>
            </Pressable>
          ) : null}
        </View>
        {profile.city || distance ? (
          <Text tone="inverse" numberOfLines={1}>
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <Text variant="caption" tone="inverse" numberOfLines={1} style={styles.interests}>
            {profile.interests.slice(0, 4).map(pretty).join("  ·  ")}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

export function pretty(slug: string): string {
  const text = slug.replace(/_/g, " ");
  return text.charAt(0).toUpperCase() + text.slice(1);
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  flex: { flex: 1 },
  tapRow: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  tap: { flex: 1 },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#FFFFFF" },
  caption: { position: "absolute", left: 0, right: 0, bottom: 0, padding: 18, gap: 4 },
  titleRow: { flexDirection: "row", alignItems: "center", gap: 10 },
  info: {
    width: 34,
    height: 34,
    borderRadius: 17,
    borderWidth: 1.5,
    borderColor: "#FFFFFF",
    alignItems: "center",
    justifyContent: "center"
  },
  interests: { opacity: 0.9 }
});
