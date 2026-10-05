import { Ionicons } from "@expo/vector-icons";
import { useEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ProfileCard } from "../api/types";
import { distanceLabel } from "../lib/labels";
import { Chip, RemoteImage, Text } from "../shared/ui";
import { radii, spacing } from "../theme";

/** Fake gradient (no native dependency): a stack of translucent layers darkening toward the bottom. */
function Scrim({ height }: { height: number }) {
  const layers = 8;
  return (
    <View pointerEvents="none" style={[styles.scrim, { height }]}>
      {Array.from({ length: layers }, (_, i) => (
        <View key={i} style={{ flex: 1, backgroundColor: `rgba(20,10,24,${((i + 1) / layers) * 0.78})` }} />
      ))}
    </View>
  );
}

/**
 * The main discovery card: swipe-independent content. Tap the left/right third to browse photos.
 * `onOpenDetails` opens the full profile.
 */
export function ProfileCardView({ profile, onOpenDetails, interactive = true }: { profile: ProfileCard; onOpenDetails?: () => void; interactive?: boolean }) {
  const [index, setIndex] = useState(0);
  const photos = profile.photos;

  useEffect(() => setIndex(0), [profile.id]);

  const go = (delta: number) => setIndex((current) => Math.max(0, Math.min(photos.length - 1, current + delta)));
  const distance = distanceLabel(profile.distanceKm);

  return (
    <View style={styles.card} testID={`card-${profile.firstName}`}>
      <RemoteImage path={photos[index]?.url} style={StyleSheet.absoluteFill} accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${profile.firstName}`} />

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}

      {interactive ? (
        <View style={styles.tapZones}>
          <Pressable style={styles.zone} onPress={() => go(-1)} accessibilityRole="button" accessibilityLabel="Photo précédente" />
          <Pressable style={styles.zone} onPress={() => go(1)} accessibilityRole="button" accessibilityLabel="Photo suivante" />
        </View>
      ) : null}

      <Scrim height={230} />
      <View style={styles.info} pointerEvents="box-none">
        <View style={styles.nameRow}>
          <Text variant="title" tone="onImage" numberOfLines={1} style={styles.name}>
            {profile.firstName}, {profile.age}
          </Text>
          {onOpenDetails ? (
            <Pressable accessibilityRole="button" accessibilityLabel={`Voir le profil de ${profile.firstName}`} testID="card-details" onPress={onOpenDetails} hitSlop={12}>
              <Ionicons name="information-circle-outline" size={30} color="#FFFFFF" />
            </Pressable>
          ) : null}
        </View>
        {profile.occupation ? (
          <Text tone="onImage" numberOfLines={1}>
            {profile.occupation}
          </Text>
        ) : null}
        {distance || profile.city ? (
          <View style={styles.row}>
            <Ionicons name="location-outline" size={16} color="#FFFFFF" />
            <Text variant="caption" tone="onImage">
              {[profile.city, distance].filter(Boolean).join(" · ")}
            </Text>
          </View>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 3).map((interest) => (
              <Chip key={interest.slug} label={interest.label} selected />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: "#2B2036" },
  dots: { position: "absolute", top: spacing.md, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "#FFFFFF" },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0 },
  info: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.lg, gap: spacing.xs },
  nameRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.sm },
  name: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: spacing.xs }
});
