import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { ProfileCard } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/palette";
import { Icon, RemoteImage, Text } from "../../ui";
import { formatDistance } from "../../utils/format";

type Props = {
  card: ProfileCard;
  onOpenDetails?: () => void;
  active?: boolean;
};

/** Full-bleed card with a tappable photo carousel and a soft bottom scrim. */
export function ProfileCardView({ card, onOpenDetails, active = true }: Props) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photo = card.photos[index] ?? card.photos[0];
  const distance = formatDistance(card.distanceKm);

  return (
    <View style={[styles.card, { backgroundColor: colors.surfaceMuted }]} testID={`card-${card.firstName}`}>
      <RemoteImage url={photo?.url} style={StyleSheet.absoluteFill} priority={active ? "high" : "normal"} accessibilityLabel={`Photo de ${card.firstName}`} />

      {card.photos.length > 1 ? (
        <View style={styles.segments} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.segment, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}

      {active && card.photos.length > 1 ? (
        <View style={styles.tapZones}>
          <Pressable
            style={styles.flex}
            accessibilityRole="button"
            accessibilityLabel="Photo précédente"
            onPress={() => setIndex((i) => Math.max(0, i - 1))}
          />
          <Pressable
            style={styles.flex}
            accessibilityRole="button"
            accessibilityLabel="Photo suivante"
            onPress={() => setIndex((i) => Math.min(card.photos.length - 1, i + 1))}
          />
        </View>
      ) : null}

      <View style={styles.scrim} pointerEvents="box-none">
        <LinearGradient
          colors={["rgba(12,8,6,0)", "rgba(12,8,6,0.35)", "rgba(12,8,6,0.82)"]}
          locations={[0, 0.4, 1]}
          style={StyleSheet.absoluteFill}
          pointerEvents="none"
        />
        <View style={styles.info} pointerEvents="box-none">
          {card.isDemo ? (
            <View style={styles.demo}>
              <Text variant="overline" tone="inverse">
                PROFIL DE DÉMONSTRATION
              </Text>
            </View>
          ) : null}
          <Pressable onPress={onOpenDetails} accessibilityRole="button" accessibilityLabel={`Voir le profil complet de ${card.firstName}`} disabled={!onOpenDetails}>
            <View style={styles.nameRow}>
              <Text variant="title" tone="inverse" numberOfLines={1} style={styles.shrink}>
                {card.firstName}, {card.age}
              </Text>
              {onOpenDetails ? <Icon name="information-circle-outline" size={24} color="#FFFFFF" /> : null}
            </View>
            {distance || card.city ? (
              <View style={styles.meta}>
                <Icon name="location-outline" size={15} color="rgba(255,255,255,0.9)" />
                <Text variant="label" tone="inverse">
                  {[distance, card.city].filter(Boolean).join(" · ")}
                </Text>
              </View>
            ) : null}
            {card.bio ? (
              <Text variant="body" tone="inverse" numberOfLines={2} style={styles.bio}>
                {card.bio}
              </Text>
            ) : null}
            {card.interests.length ? (
              <View style={styles.tags}>
                {card.interests.slice(0, 4).map((i) => (
                  <View key={i.id} style={styles.tag}>
                    <Text variant="caption" tone="inverse">
                      {i.label}
                    </Text>
                  </View>
                ))}
                {card.sharedInterests > 0 ? (
                  <View style={[styles.tag, { backgroundColor: colors.primary }]}>
                    <Text variant="caption" style={{ color: colors.onPrimary }}>
                      {card.sharedInterests} en commun
                    </Text>
                  </View>
                ) : null}
              </View>
            ) : null}
          </Pressable>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  flex: { flex: 1 },
  shrink: { flexShrink: 1 },
  segments: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  segment: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, bottom: "38%", flexDirection: "row" },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0, top: "40%", justifyContent: "flex-end" },
  info: { padding: spacing.lg, gap: 6 },
  demo: { alignSelf: "flex-start", backgroundColor: "rgba(0,0,0,0.45)", paddingHorizontal: 8, paddingVertical: 3, borderRadius: 6 },
  nameRow: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  bio: { marginTop: 4, opacity: 0.95 },
  tags: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 6 },
  tag: { backgroundColor: "rgba(255,255,255,0.22)", paddingHorizontal: 10, paddingVertical: 4, borderRadius: radii.pill }
});
