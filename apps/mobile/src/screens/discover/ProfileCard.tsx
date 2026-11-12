import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { PublicProfile } from "../../api/types";
import { radii, spacing, useTheme } from "../../theme";
import { Icon, IconButton, Photo, Text } from "../../ui";
import { distanceLabel, nameAge } from "../../utils/format";

/** A discovery card: tap left/right edges to browse photos, info opens the full profile. */
export function ProfileCard({ profile, onOpen, active }: { profile: PublicProfile; onOpen: () => void; active: boolean }) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const photo = photos[Math.min(index, photos.length - 1)];
  const distance = distanceLabel(profile.distanceKm);

  return (
    <View style={[styles.card, { backgroundColor: colors.surfaceMuted }]}>
      <Photo url={photo?.url} style={StyleSheet.absoluteFill} priority={active ? "high" : "normal"} accessibilityLabel={`Photo de ${profile.firstName}`} />
      {photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" style={styles.tap} onPress={() => setIndex((i) => Math.max(0, i - 1))} />
        <Pressable accessibilityLabel="Photo suivante" style={styles.tap} onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))} />
      </View>
      <LinearGradient
        colors={["transparent", "rgba(10,6,4,0.35)", "rgba(10,6,4,0.78)"]}
        locations={[0, 0.35, 1]}
        style={styles.scrimWrap}
        pointerEvents="box-none"
      >
        <View style={styles.info}>
          <View style={styles.infoRow}>
            <View style={styles.infoText}>
              <Text variant="title" serif tone="onPhoto" numberOfLines={1}>
                {nameAge(profile.firstName, profile.age)}
              </Text>
              {profile.occupation ? (
                <Text tone="onPhoto" numberOfLines={1} style={styles.dim}>
                  {profile.occupation}
                </Text>
              ) : null}
              {distance || profile.city ? (
                <View style={styles.meta}>
                  <Icon name="location-outline" size={15} color="#FFFFFF" />
                  <Text variant="caption" tone="onPhoto" style={styles.dim}>
                    {[profile.city, distance].filter(Boolean).join(" · ")}
                  </Text>
                </View>
              ) : null}
            </View>
            <IconButton icon="chevron-up" label={`Voir le profil de ${profile.firstName}`} tone="onPhoto" onPress={onOpen} />
          </View>
          {profile.interests.length ? (
            <View style={styles.chips}>
              {profile.interests.slice(0, 4).map((i) => (
                <View key={i.id} style={styles.chip}>
                  <Text variant="caption" tone="onPhoto">
                    {i.label}
                  </Text>
                </View>
              ))}
            </View>
          ) : null}
        </View>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "35%" },
  tap: { flex: 1 },
  scrimWrap: { position: "absolute", left: 0, right: 0, bottom: 0 },
  info: { paddingHorizontal: spacing.lg, paddingBottom: spacing.lg, paddingTop: spacing.xxxl * 1.5, gap: spacing.sm },
  infoRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm },
  infoText: { flex: 1, gap: 2 },
  dim: { opacity: 0.9 },
  meta: { flexDirection: "row", alignItems: "center", gap: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6 },
  chip: { borderRadius: radii.pill, paddingHorizontal: 10, paddingVertical: 4, backgroundColor: "rgba(255,255,255,0.2)" }
});
