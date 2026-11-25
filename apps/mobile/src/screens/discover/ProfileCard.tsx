import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { PublicProfile } from "../../api/types";
import { colors, radii, spacing } from "../../theme";
import { Chip, RemoteImage, Text } from "../../ui";
import { formatDistance } from "../../utils/format";

type ProfileCardProps = {
  profile: PublicProfile;
  onOpen: () => void;
  active: boolean;
};

/** Discovery card: photo carousel (tap left/right), name, age, distance and interests. */
export function ProfileCard({ profile, onOpen, active }: ProfileCardProps) {
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const photo = photos[index] ?? photos[0];
  const distance = formatDistance(profile.distanceKm);
  const common = new Set(profile.commonInterestIds);
  const interests = [...profile.interests].sort((a, b) => Number(common.has(b.id)) - Number(common.has(a.id))).slice(0, 4);

  return (
    <View style={styles.card} testID={active ? "discover-card" : undefined}>
      <RemoteImage
        path={photo?.url}
        style={StyleSheet.absoluteFillObject}
        priority={active ? "high" : "low"}
        accessibilityLabel={`Photo de ${profile.firstName}`}
      />

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, i === index && styles.dotActive]} />
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

      {profile.isDemo ? (
        <View style={styles.demo}>
          <Text variant="caption" color="onPrimary">
            Profil de démonstration
          </Text>
        </View>
      ) : null}

      <View style={styles.info} pointerEvents="box-none">
        <View style={styles.shade1} pointerEvents="none" />
        <View style={styles.shade2} pointerEvents="none" />
        <Pressable
          onPress={onOpen}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${profile.firstName}`}
          style={styles.infoContent}
          testID={active ? "discover-open-profile" : undefined}
        >
          <View style={styles.nameRow}>
            <Text variant="title" style={styles.shadowText} numberOfLines={1}>
              {profile.firstName}, {profile.age}
            </Text>
            <Ionicons name="information-circle-outline" size={26} color={colors.text} />
          </View>
          {profile.jobTitle || distance ? (
            <Text variant="small" color="textMuted" style={styles.shadowText}>
              {[profile.jobTitle, distance].filter(Boolean).join(" · ")}
            </Text>
          ) : null}
          {interests.length ? (
            <View style={styles.chips}>
              {interests.map((i) => (
                <Chip key={i.id} label={i.label} highlighted={common.has(i.id)} />
              ))}
            </View>
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
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  flex: {
    flex: 1
  },
  dots: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: 4
  },
  dot: {
    flex: 1,
    height: 3,
    borderRadius: 2,
    backgroundColor: "rgba(255,255,255,0.35)"
  },
  dotActive: {
    backgroundColor: colors.white
  },
  tapZones: {
    ...StyleSheet.absoluteFillObject,
    bottom: "35%",
    flexDirection: "row"
  },
  demo: {
    position: "absolute",
    top: spacing.lg,
    right: spacing.md,
    backgroundColor: colors.gold,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0
  },
  shade1: {
    ...StyleSheet.absoluteFillObject,
    top: -60,
    backgroundColor: "rgba(21,17,26,0.35)"
  },
  shade2: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: "rgba(21,17,26,0.55)"
  },
  infoContent: {
    padding: spacing.lg,
    gap: spacing.xs
  },
  nameRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  shadowText: {
    textShadowColor: "rgba(0,0,0,0.6)",
    textShadowRadius: 6,
    flexShrink: 1
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs,
    marginTop: spacing.xxs
  }
});
