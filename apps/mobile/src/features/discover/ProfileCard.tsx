import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { mediaUrl } from "../../api/client";
import type { DiscoverCard } from "../../api/types";
import { colors, radii, spacing } from "../../theme";
import { Text } from "../../ui";
import { distanceLabel } from "../../utils/format";

type Props = {
  card: DiscoverCard;
  onOpenDetails: () => void;
};

/** Full-bleed card with tap-to-browse photos (left/right halves) and key facts. */
export function ProfileCard({ card, onOpenDetails }: Props) {
  const [index, setIndex] = useState(0);
  const photos = card.photos;
  const photo = photos[index] ?? photos[0];
  const distance = distanceLabel(card.distanceKm);

  return (
    <View style={styles.card} testID={`card-${card.userId}`} accessibilityLabel={`${card.firstName}, ${card.age} ans`}>
      {photo ? (
        <Image
          source={{ uri: mediaUrl(photo.url) }}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          cachePolicy="memory-disk"
          transition={120}
          accessibilityIgnoresInvertColors
        />
      ) : null}
      {photos.length > 1 ? (
        <>
          <View style={styles.bars}>
            {photos.map((p, i) => (
              <View key={p.id} style={[styles.bar, i === index ? styles.barActive : null]} />
            ))}
          </View>
          <View style={styles.tapZones}>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Photo précédente"
              style={styles.tapZone}
              onPress={() => setIndex((i) => Math.max(0, i - 1))}
            />
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Photo suivante"
              style={styles.tapZone}
              onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
            />
          </View>
        </>
      ) : null}
      {card.isDemo ? (
        <View style={styles.demo}>
          <Text variant="caption" tone="inverse">
            Profil de démonstration
          </Text>
        </View>
      ) : null}

      <View style={styles.infoShade} pointerEvents="box-none">
        <Pressable
          onPress={onOpenDetails}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${card.firstName}`}
          style={styles.info}
        >
          <View style={styles.nameRow}>
            <Text variant="title" style={styles.name} numberOfLines={1}>
              {card.firstName}
            </Text>
            <Text variant="title" tone="muted" style={styles.age}>
              {card.age}
            </Text>
            <View style={styles.flex} />
            <Ionicons name="information-circle" size={28} color={colors.text} />
          </View>
          {card.jobTitle ? (
            <View style={styles.meta}>
              <Ionicons name="briefcase-outline" size={15} color={colors.textMuted} />
              <Text tone="muted" variant="label" numberOfLines={1}>
                {card.jobTitle}
              </Text>
            </View>
          ) : null}
          {distance ? (
            <View style={styles.meta}>
              <Ionicons name="location-outline" size={15} color={colors.textMuted} />
              <Text tone="muted" variant="label">
                {distance}
                {card.city ? ` · ${card.city}` : ""}
              </Text>
            </View>
          ) : null}
          {card.sharedInterests > 0 ? (
            <View style={styles.meta}>
              <Ionicons name="sparkles" size={15} color={colors.gold} />
              <Text tone="gold" variant="label">
                {card.sharedInterests} centre{card.sharedInterests > 1 ? "s" : ""} d’intérêt en commun
              </Text>
            </View>
          ) : null}
          {card.bio ? (
            <Text tone="muted" numberOfLines={2} style={styles.bio}>
              {card.bio}
            </Text>
          ) : null}
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    flex: 1,
    borderRadius: radii.xl,
    overflow: "hidden",
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.border
  },
  bars: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: spacing.xs,
    zIndex: 2
  },
  bar: {
    flex: 1,
    height: 3,
    borderRadius: 2,
    backgroundColor: "rgba(255,255,255,0.35)"
  },
  barActive: {
    backgroundColor: colors.white
  },
  tapZones: {
    ...StyleSheet.absoluteFillObject,
    bottom: "35%",
    flexDirection: "row",
    zIndex: 1
  },
  tapZone: {
    flex: 1
  },
  demo: {
    position: "absolute",
    top: spacing.xl,
    left: spacing.md,
    backgroundColor: colors.gold,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    zIndex: 2
  },
  infoShade: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    paddingTop: spacing.xxxl,
    backgroundColor: "rgba(21,16,26,0.78)",
    zIndex: 2
  },
  info: {
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing.lg,
    paddingTop: spacing.sm,
    gap: spacing.xs,
    marginTop: -spacing.xxxl + spacing.md
  },
  nameRow: {
    flexDirection: "row",
    alignItems: "baseline",
    gap: spacing.sm
  },
  name: {
    flexShrink: 1
  },
  age: {
    fontWeight: "400"
  },
  flex: {
    flex: 1
  },
  meta: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  bio: {
    marginTop: spacing.xs
  }
});
