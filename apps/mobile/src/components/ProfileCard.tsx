import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { Chip, IconButton, Photo, Text, colors, radii, spacing } from "../shared/ui";
import { formatDistance } from "../utils/format";

type ProfileCardProps = {
  profile: PublicProfile;
  onOpen: () => void;
};

/** Discovery card: tap left/right to browse photos, "i" for the full profile. */
export function ProfileCard({ onOpen, profile }: ProfileCardProps) {
  const [index, setIndex] = useState(0);
  const photo = profile.photos[index] ?? profile.photos[0];
  const distance = formatDistance(profile.distanceKm);
  const count = profile.photos.length;

  return (
    <View style={styles.card} testID={`card-${profile.userId}`}>
      <Photo
        accessibilityLabel={`Photo ${index + 1} de ${profile.firstName}`}
        placeholderLabel={profile.firstName.slice(0, 1)}
        priority="high"
        style={StyleSheet.absoluteFill}
        url={photo?.url}
      />
      {count > 1 ? (
        <View style={styles.bars}>
          {profile.photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, i === index ? styles.barActive : null]} />
          ))}
        </View>
      ) : null}
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.tapZone} />
        <Pressable accessibilityLabel="Photo suivante" onPress={() => setIndex((i) => Math.min(count - 1, i + 1))} style={styles.tapZone} />
      </View>
      <View pointerEvents="box-none" style={styles.info}>
        <View pointerEvents="box-none" style={styles.titleRow}>
          <View style={styles.titleText}>
            <Text numberOfLines={1} variant="title">
              {profile.firstName},{" "}
              <Text variant="title" weight="regular">
                {profile.age}
              </Text>
            </Text>
            {distance || profile.city ? (
              <Text tone="muted" variant="label">
                {[profile.city, distance].filter(Boolean).join(" · ")}
              </Text>
            ) : null}
          </View>
          <IconButton
            accessibilityLabel={`Voir le profil de ${profile.firstName}`}
            glyph="i"
            onPress={onOpen}
            size={40}
            testID="card-open"
          />
        </View>
        {profile.bio ? (
          <Text numberOfLines={2} style={styles.bio}>
            {profile.bio}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 4).map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
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
    top: spacing.md,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: 4,
    zIndex: 2
  },
  bar: {
    flex: 1,
    height: 3,
    borderRadius: 2,
    backgroundColor: "rgba(255,255,255,0.3)"
  },
  barActive: {
    backgroundColor: "#FFFFFF"
  },
  tapZones: {
    ...StyleSheet.absoluteFillObject,
    flexDirection: "row",
    bottom: "40%"
  },
  tapZone: {
    flex: 1
  },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxxl,
    gap: spacing.sm,
    backgroundColor: "rgba(12, 9, 8, 0.78)"
  },
  titleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  titleText: {
    flex: 1
  },
  bio: {
    color: colors.text
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
