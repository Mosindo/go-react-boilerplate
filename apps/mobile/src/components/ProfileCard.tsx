import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { PublicProfile } from "../api/types";
import { formatDistance, nameWithAge } from "../lib/format";
import { Badge, Text, radii, spacing } from "../shared/ui";
import { PhotoImage } from "./PhotoImage";

type ProfileCardProps = {
  profile: PublicProfile;
  onOpenProfile: () => void;
};

/** Full-bleed photo card used by the discovery deck. Tap the left/right side to browse photos. */
export function ProfileCard({ profile, onOpenProfile }: ProfileCardProps) {
  const [index, setIndex] = useState(0);
  useEffect(() => setIndex(0), [profile.userId]);

  const photos = profile.photos;
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];
  const distance = formatDistance(profile.distanceKm);
  const where = [profile.city, distance].filter(Boolean).join(" · ");
  const shared = profile.interests.filter((interest) => interest.shared);
  const chips = (shared.length > 0 ? shared : profile.interests).slice(0, 3);

  return (
    <View style={styles.root}>
      <PhotoImage accessibilityLabel={`Photo de ${profile.firstName}`} path={current?.url} style={StyleSheet.absoluteFill} />

      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.bars}>
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}

      <View style={styles.zones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.zoneLeft} />
        <Pressable
          accessibilityLabel="Photo suivante"
          onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
          style={styles.zoneRight}
        />
      </View>

      <View pointerEvents="box-none" style={styles.scrim}>
        <View style={styles.titleRow}>
          <View style={styles.titleCopy}>
            <Text numberOfLines={1} style={styles.white} variant="title">
              {nameWithAge(profile.firstName, profile.age)}
            </Text>
            {where ? (
              <View style={styles.whereRow}>
                <Ionicons color="#FFFFFF" name="location-outline" size={14} />
                <Text numberOfLines={1} style={styles.white} variant="label">
                  {where}
                </Text>
              </View>
            ) : null}
          </View>
          <Pressable accessibilityLabel={`Voir le profil de ${profile.firstName}`} hitSlop={12} onPress={onOpenProfile} style={styles.info}>
            <Ionicons color="#FFFFFF" name="information" size={22} />
          </Pressable>
        </View>
        {chips.length > 0 ? (
          <View style={styles.chips}>
            {chips.map((interest) => (
              <Badge key={interest.id} label={interest.label} tone={interest.shared ? "primary" : "neutral"} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: "#2A2220" },
  bars: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  zoneLeft: { flex: 1 },
  zoneRight: { flex: 2 },
  scrim: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    gap: spacing.sm,
    backgroundColor: "rgba(20, 12, 8, 0.55)"
  },
  titleRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.md },
  titleCopy: { flex: 1, gap: 2 },
  white: { color: "#FFFFFF" },
  whereRow: { flexDirection: "row", alignItems: "center", gap: 4 },
  info: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: "rgba(255,255,255,0.25)",
    alignItems: "center",
    justifyContent: "center"
  },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
