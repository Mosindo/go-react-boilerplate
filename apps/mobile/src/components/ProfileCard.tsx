import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { PublicProfile } from "../api/types";
import { formatDistance } from "../shared/format";
import { Chip, PhotoImage, Text, radii, spacing } from "../shared/ui";

type ProfileCardProps = {
  profile: PublicProfile;
  onOpen?: () => void;
  /** Photo tapping is disabled while the card is being dragged. */
  interactive?: boolean;
};

/** Full-bleed photo card: tap the left/right edge to flip photos, info button for the full profile. */
export function ProfileCard({ profile, onOpen, interactive = true }: ProfileCardProps) {
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const photos = profile.photos;
  const distance = formatDistance(profile.distanceKm);

  useEffect(() => setIndex(0), [profile.id]);

  const flip = (locationX: number) => {
    if (!interactive || photos.length < 2 || width === 0) return;
    if (locationX < width / 3) setIndex((i) => Math.max(0, i - 1));
    else if (locationX > (width * 2) / 3) setIndex((i) => Math.min(photos.length - 1, i + 1));
    else onOpen?.();
  };

  return (
    <View onLayout={(e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width)} style={styles.card}>
      <Pressable
        accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${profile.firstName}`}
        accessibilityRole="imagebutton"
        onPress={(e) => flip(e.nativeEvent.locationX)}
        style={StyleSheet.absoluteFill}
      >
        <PhotoImage path={photos[index]?.url} style={StyleSheet.absoluteFill} />
      </Pressable>

      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.bars}>
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}

      <View pointerEvents="box-none" style={styles.scrim}>
        <View pointerEvents="none" style={styles.titleRow}>
          <Text style={styles.onPhoto} variant="title">
            {profile.firstName}, {profile.age}
          </Text>
        </View>
        {profile.occupation ? (
          <Text pointerEvents="none" style={styles.onPhotoSoft} weight="medium">
            {profile.occupation}
          </Text>
        ) : null}
        {distance || profile.city ? (
          <View pointerEvents="none" style={styles.meta}>
            <Ionicons color="#FFFFFF" name="location-outline" size={15} />
            <Text style={styles.onPhotoSoft} variant="bodySmall">
              {[profile.city, distance].filter(Boolean).join(" · ")}
            </Text>
          </View>
        ) : null}
        <View pointerEvents="none" style={styles.chips}>
          {profile.interests.slice(0, 3).map((interest) => (
            <Chip highlighted={false} key={interest.slug} label={interest.label} />
          ))}
        </View>
        {onOpen ? (
          <Pressable accessibilityLabel={`Voir le profil de ${profile.firstName}`} accessibilityRole="button" hitSlop={10} onPress={onOpen} style={styles.info}>
            <Ionicons color="#FFFFFF" name="information-circle-outline" size={30} />
          </Pressable>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: "#000" },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  scrim: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxxl,
    gap: spacing.xs,
    backgroundColor: "rgba(20, 8, 16, 0.5)"
  },
  titleRow: { paddingRight: 40 },
  onPhoto: { color: "#FFFFFF" },
  onPhotoSoft: { color: "rgba(255,255,255,0.92)" },
  meta: { flexDirection: "row", alignItems: "center", gap: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: spacing.xs },
  info: { position: "absolute", right: spacing.lg, top: spacing.xl }
});
