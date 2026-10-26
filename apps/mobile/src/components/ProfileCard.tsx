import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api";
import { distanceLabel } from "../lib/format";
import { radius, spacing } from "../theme";
import { Chip } from "./Chip";
import { Photo } from "./Photo";
import { Text } from "./Text";

type Props = { profile: PublicProfile; onInfo?: () => void };

/** Discovery card: tap the left/right side of the photo to browse pictures. */
export function ProfileCard({ profile, onInfo }: Props) {
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const current = photos[index] ?? photos[0];
  const distance = distanceLabel(profile.distanceKm);

  const step = (delta: number) => setIndex((i) => Math.min(photos.length - 1, Math.max(0, i + delta)));

  return (
    <View style={styles.card} testID="profile-card">
      <Photo
        url={current?.url}
        label={`Photo ${index + 1} de ${profile.firstName}`}
        style={StyleSheet.absoluteFill as never}
      />
      <View style={styles.tapRow}>
        <Pressable style={styles.tap} onPress={() => step(-1)} accessibilityLabel="Photo précédente" />
        <Pressable style={styles.tap} onPress={() => step(1)} accessibilityLabel="Photo suivante" />
      </View>
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.footer} pointerEvents="box-none">
        <Pressable
          onPress={onInfo}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${profile.firstName}`}
        >
          <Text variant="title" color="#FFFFFF" style={styles.shadow}>
            {profile.firstName}, {profile.age}
          </Text>
          {profile.jobTitle ? (
            <Text color="#FFFFFF" style={styles.shadow}>
              {profile.jobTitle}
            </Text>
          ) : null}
          <Text variant="caption" color="#FFFFFF" style={styles.shadow}>
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        </Pressable>
        {profile.interests.length ? (
          <View style={styles.chips} pointerEvents="none">
            {profile.interests.slice(0, 3).map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radius.lg, overflow: "hidden", backgroundColor: "#2A1F2D" },
  tapRow: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 120 },
  tap: { flex: 1 },
  dots: {
    position: "absolute",
    top: spacing.md,
    left: spacing.lg,
    right: spacing.lg,
    flexDirection: "row",
    gap: 4
  },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#FFFFFF" },
  footer: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    gap: 4,
    backgroundColor: "rgba(24, 14, 28, 0.55)"
  },
  chips: { flexDirection: "row", gap: spacing.xs, marginTop: spacing.sm, flexWrap: "wrap" },
  shadow: {
    textShadowColor: "rgba(0,0,0,0.5)",
    textShadowRadius: 4,
    textShadowOffset: { width: 0, height: 1 }
  }
});
