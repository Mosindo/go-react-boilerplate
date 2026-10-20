import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { Profile } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { radii, spacing } from "../../theme";
import { RemoteImage, Text } from "../../ui";

export function nextPhotoIndex(current: number, count: number, direction: 1 | -1): number {
  if (count <= 1) return 0;
  return Math.min(count - 1, Math.max(0, current + direction));
}

export function ProfileCard({ profile, onOpen }: { profile: Profile; onOpen: () => void }) {
  const [index, setIndex] = useState(0);
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const distance = formatDistance(profile.distanceKm);

  return (
    <View style={styles.card} accessible accessibilityLabel={`${profile.firstName}, ${profile.age}`}>
      <Pressable
        style={StyleSheet.absoluteFill}
        accessibilityLabel="Next photo"
        onPress={(e) => {
          const leftHalf = e.nativeEvent.locationX < 120;
          setIndex((i) => nextPhotoIndex(i, photos.length, leftHalf ? -1 : 1));
        }}
      >
        <RemoteImage url={photos[index]?.url ?? null} style={StyleSheet.absoluteFill} />
      </Pressable>
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <LinearGradient
        colors={["transparent", "rgba(20,8,16,0.85)"]}
        style={styles.gradient}
        pointerEvents="box-none"
      >
        <View style={styles.row}>
          <View style={styles.flex} pointerEvents="none">
            <Text variant="title" style={styles.white}>
              {profile.firstName}, {profile.age}
            </Text>
            <Text style={styles.white}>{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
            {profile.interests.length > 0 ? (
              <Text variant="caption" style={styles.white} numberOfLines={1}>
                {profile.interests
                  .slice(0, 3)
                  .map((i) => i.label)
                  .join(" • ")}
              </Text>
            ) : null}
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Open ${profile.firstName}'s full profile`}
            onPress={onOpen}
            style={styles.info}
            testID="open-profile"
          >
            <Text variant="label" style={styles.white}>
              i
            </Text>
          </Pressable>
        </View>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.lg, overflow: "hidden", backgroundColor: "#3b2a33" },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, paddingTop: 64 },
  row: { flexDirection: "row", alignItems: "flex-end" },
  flex: { flex: 1 },
  white: { color: "#FFFFFF" },
  info: {
    width: 40,
    height: 40,
    borderRadius: 20,
    borderWidth: 1.5,
    borderColor: "#FFFFFF",
    alignItems: "center",
    justifyContent: "center"
  },
  dots: {
    position: "absolute",
    top: spacing.md,
    left: spacing.lg,
    right: spacing.lg,
    flexDirection: "row",
    justifyContent: "center"
  },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#FFFFFF", marginHorizontal: 2 }
});
