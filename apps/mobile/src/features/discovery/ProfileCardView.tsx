import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ProfileCard } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, shadow, spacing } from "../../theme/tokens";
import { Chip, Icon, IconButton, RemoteImage, Text } from "../../ui";

type Props = { card: ProfileCard; onOpen: () => void; active?: boolean };

/** Photo-first card: tap the left/right side of the photo to browse, "i" opens the full profile. */
export function ProfileCardView({ card, onOpen, active = true }: Props) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = card.photos;
  const photo = photos[Math.min(index, photos.length - 1)];
  const distance = formatDistance(card.distanceKm);

  return (
    <View style={[styles.card, shadow, { backgroundColor: colors.surfaceMuted }]} testID={`card-${card.userId}`}>
      <RemoteImage
        path={photo?.url}
        style={StyleSheet.absoluteFill}
        priority={active ? "high" : "low"}
        accessibilityLabel={`Photo de ${card.firstName}`}
      />
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      {active && photos.length > 1 ? (
        <View style={styles.tapZones}>
          <Pressable style={styles.flex} accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} />
          <Pressable
            style={styles.flex}
            accessibilityLabel="Photo suivante"
            onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
          />
        </View>
      ) : null}
      <View style={[styles.info, { backgroundColor: colors.scrim }]} pointerEvents="box-none">
        <View style={styles.titleRow}>
          <View style={styles.flex}>
            <Text variant="title" tone="onPhoto" numberOfLines={1}>
              {card.firstName}, {card.age}
            </Text>
            {card.jobTitle ? (
              <Text tone="onPhoto" numberOfLines={1} style={styles.soft}>
                {card.jobTitle}
              </Text>
            ) : null}
            {distance || card.city ? (
              <View style={styles.meta}>
                <Icon name="location-outline" size={15} color="#FFFFFF" />
                <Text variant="caption" tone="onPhoto" style={styles.soft}>
                  {[distance, card.city].filter(Boolean).join(" · ")}
                </Text>
              </View>
            ) : null}
          </View>
          {active ? (
            <IconButton
              icon="information"
              label={`Voir le profil de ${card.firstName}`}
              tone="default"
              size={40}
              onPress={onOpen}
              testID="card-open"
            />
          ) : null}
        </View>
        {card.interests.length > 0 ? (
          <View style={styles.chips}>
            {card.interests.slice(0, 4).map((i) => (
              <Chip key={i.id} label={i.label} variant="onPhoto" />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  dots: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  flex: { flex: 1 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, gap: spacing.sm },
  titleRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  soft: { opacity: 0.9 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6 }
});
