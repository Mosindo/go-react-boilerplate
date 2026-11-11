import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Card } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Photo, Text } from "../../ui";
import { formatDistance } from "../../utils/format";

// Stepped scrim that approximates a gradient without a native dependency.
const SHADE_STEPS = [0.04, 0.1, 0.18, 0.28, 0.38, 0.48, 0.56, 0.62, 0.66, 0.68];

type Props = {
  card: Card;
  onOpenDetails?: () => void;
  interactive?: boolean;
};

/** Full-bleed profile card. Tapping the left/right half browses photos. */
export function ProfileCard({ card, onOpenDetails, interactive = true }: Props) {
  const { colors, radii } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = card.photos;
  const photo = photos[Math.min(index, photos.length - 1)];
  const distance = formatDistance(card.distanceKm);
  const shared = card.interests.filter((i) => i.shared);

  return (
    <View style={[styles.card, { borderRadius: radii.xl, backgroundColor: colors.surfaceSunken }]}>
      <Photo
        url={photo?.url}
        style={StyleSheet.absoluteFillObject}
        priority="high"
        accessibilityLabel={`Photo de ${card.firstName}`}
      />
      {interactive && photos.length > 1 ? (
        <View style={StyleSheet.absoluteFill} pointerEvents="box-none">
          <View style={styles.tapZones}>
            <Pressable
              style={styles.flex}
              accessibilityLabel="Photo précédente"
              onPress={() => setIndex((i) => Math.max(0, i - 1))}
            />
            <Pressable
              style={styles.flex}
              accessibilityLabel="Photo suivante"
              onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
            />
          </View>
        </View>
      ) : null}
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[styles.dot, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]}
            />
          ))}
        </View>
      ) : null}
      <View style={styles.shade} pointerEvents="none">
        {SHADE_STEPS.map((opacity) => (
          <View key={opacity} style={[styles.shadeStep, { opacity }]} />
        ))}
      </View>
      <Pressable
        style={styles.info}
        onPress={onOpenDetails}
        disabled={!onOpenDetails}
        accessibilityRole="button"
        accessibilityLabel={`Voir le profil complet de ${card.firstName}, ${card.age} ans`}
        testID={`card-${card.userId}`}
      >
        <View style={styles.titleRow}>
          <Text variant="title" tone="inverse" style={styles.flex} numberOfLines={1}>
            {card.firstName}, {card.age}
          </Text>
          {onOpenDetails ? <Ionicons name="information-circle-outline" size={28} color="#FFFFFF" /> : null}
        </View>
        {card.jobTitle ? (
          <Text tone="inverse" numberOfLines={1}>
            {card.jobTitle}
          </Text>
        ) : null}
        {distance || card.city ? (
          <View style={styles.meta}>
            <Ionicons name="location-outline" size={15} color="rgba(255,255,255,0.85)" />
            <Text tone="inverse" variant="caption" style={{ opacity: 0.85 }}>
              {[distance, card.city].filter(Boolean).join(" · ")}
            </Text>
          </View>
        ) : null}
        {shared.length > 0 ? (
          <View style={styles.chips}>
            {shared.slice(0, 3).map((i) => (
              <View key={i.slug} style={styles.chip}>
                <Text variant="caption" tone="inverse" weight="600">
                  {i.label}
                </Text>
              </View>
            ))}
          </View>
        ) : null}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, overflow: "hidden" },
  flex: { flex: 1 },
  tapZones: { flex: 1, flexDirection: "row", marginBottom: 140 },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  shade: { position: "absolute", left: 0, right: 0, bottom: 0, height: "45%" },
  shadeStep: { flex: 1, backgroundColor: "#140E11" },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: 20, gap: 4 },
  titleRow: { flexDirection: "row", alignItems: "center", gap: 8 },
  meta: { flexDirection: "row", alignItems: "center", gap: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 6 },
  chip: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 999, backgroundColor: "rgba(255,255,255,0.2)" }
});
