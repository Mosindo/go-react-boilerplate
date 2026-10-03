import React, { useState } from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import type { PublicProfile } from "../api/types";
import { formatDistance } from "../lib/format";
import { Text, colors, radii, spacing } from "../shared/ui";
import { Chip } from "./Chip";
import { PhotoImage } from "./PhotoImage";

type Props = { profile: PublicProfile };

/** Full profile: tappable photo carousel followed by the details people chose to share. */
export function ProfileView({ profile }: Props) {
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const count = profile.photos.length;
  const photo = profile.photos[Math.min(index, Math.max(count - 1, 0))];
  const distance = formatDistance(profile.distanceKm);

  const step = (delta: number) => {
    if (count > 1) {
      setIndex((i) => (i + delta + count) % count);
    }
  };

  return (
    <View style={styles.root}>
      <View onLayout={(e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width)} style={[styles.gallery, { height: width * 1.25 }]}>
        <PhotoImage accessibilityLabel={`${profile.firstName}, photo ${index + 1} of ${count}`} fallbackLabel={profile.firstName} path={photo?.url} style={StyleSheet.absoluteFill} />
        {count > 1 ? (
          <>
            <Pressable accessibilityLabel="Previous photo" accessibilityRole="button" onPress={() => step(-1)} style={[styles.zone, styles.zoneLeft]} />
            <Pressable accessibilityLabel="Next photo" accessibilityRole="button" onPress={() => step(1)} style={[styles.zone, styles.zoneRight]} />
            <View pointerEvents="none" style={styles.dots}>
              {profile.photos.map((p, i) => (
                <View key={p.id} style={[styles.dot, i === index ? styles.dotActive : null]} />
              ))}
            </View>
          </>
        ) : null}
      </View>
      <View style={styles.details}>
        <Text variant="title" weight="bold">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city || distance ? (
          <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
        {profile.bio ? <Text>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={styles.wrap}>
            {profile.interests.map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.lg },
  gallery: { width: "100%", borderRadius: radii.xl, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  zone: { position: "absolute", top: 0, bottom: 0, width: "40%" },
  zoneLeft: { left: 0 },
  zoneRight: { right: 0 },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: spacing.xs },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.45)" },
  dotActive: { backgroundColor: "#ffffff" },
  details: { gap: spacing.sm, paddingHorizontal: spacing.xs },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
