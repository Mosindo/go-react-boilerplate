import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Card } from "../api/types";
import { formatDistance } from "../lib/format";
import { Text, colors, radii, spacing } from "../shared/ui";
import { Chip } from "./Chip";
import { RemotePhoto } from "./RemotePhoto";

type ProfileCardViewProps = {
  card: Card;
  /** Show bio and interests under the photo (full profile) or only the overlay (deck). */
  detailed?: boolean;
  onOpenProfile?: () => void;
};

/** Photo pager (tap left/right) with name, age and distance overlay. */
export function ProfileCardView({ card, detailed = false, onOpenProfile }: ProfileCardViewProps) {
  const [index, setIndex] = useState(0);
  const count = card.photos.length;
  const current = card.photos[Math.min(index, Math.max(count - 1, 0))];
  const distance = formatDistance(card.distanceKm);

  return (
    <View style={styles.wrap}>
      <View style={styles.photoBox}>
        <RemotePhoto accessibilityLabel={`Photo de ${card.firstName}`} style={StyleSheet.absoluteFill} url={current?.url} />
        <View style={styles.tapRow}>
          <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.tapZone} />
          <Pressable
            accessibilityLabel="Photo suivante"
            onPress={() => setIndex((i) => Math.min(count - 1, i + 1))}
            style={styles.tapZone}
          />
        </View>
        {count > 1 ? (
          <View pointerEvents="none" style={styles.dots}>
            {card.photos.map((p, i) => (
              <View key={p.id} style={[styles.dot, i === index ? styles.dotActive : null]} />
            ))}
          </View>
        ) : null}
        <View pointerEvents="none" style={styles.gradient} />
        <Pressable accessibilityRole="button" disabled={!onOpenProfile} onPress={onOpenProfile} style={styles.overlay}>
          <Text style={styles.name} tone="inverse" variant="title" weight="bold">
            {card.firstName}, {card.age}
          </Text>
          <Text tone="inverse" variant="label">
            {[card.city, distance].filter(Boolean).join(" · ")}
          </Text>
        </Pressable>
      </View>
      {detailed ? (
        <View style={styles.details}>
          {card.bio ? <Text>{card.bio}</Text> : null}
          {card.interests.length > 0 ? (
            <View style={styles.chips}>
              {card.interests.map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
          ) : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { width: "100%" },
  photoBox: { width: "100%", aspectRatio: 3 / 4, borderRadius: radii.xl, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  tapRow: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  tapZone: { flex: 1 },
  dots: { position: "absolute", top: spacing.md, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.45)" },
  dotActive: { backgroundColor: "#ffffff" },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, height: 140, backgroundColor: "rgba(43,27,36,0.45)" },
  overlay: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.lg },
  name: { fontSize: 30, lineHeight: 36 },
  details: { paddingTop: spacing.lg, gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
