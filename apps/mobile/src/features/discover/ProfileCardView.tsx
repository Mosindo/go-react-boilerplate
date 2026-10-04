import { LinearGradient } from "expo-linear-gradient";
import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import type { PublicProfile } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { nextPhotoIndex, tapZone } from "../../lib/swipe";
import { radii, spacing, useColors } from "../../theme";
import { Chip, Icon, ProfilePhoto, Text } from "../../ui";

type Props = {
  profile: PublicProfile;
  onOpen: () => void;
};

/** Full-bleed card: photo carousel (tap left/right), gradient, name/age, distance, interests, bio preview. */
export function ProfileCardView({ profile, onOpen }: Props) {
  const colors = useColors();
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const userId = profile.userId;
  useEffect(() => setIndex(0), [userId]);

  const distance = formatDistance(profile.distanceKm);
  const shownInterests = profile.interests.slice(0, 3);

  return (
    <View
      onLayout={(event: LayoutChangeEvent) => setWidth(event.nativeEvent.layout.width)}
      style={[styles.card, { backgroundColor: colors.surfaceMuted }]}
    >
      <ProfilePhoto
        name={profile.displayName}
        photo={photos[index] ?? null}
        size="full"
        style={StyleSheet.absoluteFill}
      />

      {/* Photo carousel: left half = previous, right half = next. */}
      <Pressable
        accessibilityHint="Double tap to see the next photo"
        accessibilityLabel={`Photo ${index + 1} of ${Math.max(photos.length, 1)}. Tap the left side for the previous photo and the right side for the next.`}
        accessibilityRole="button"
        onPress={(event) =>
          setIndex((current) => nextPhotoIndex(current, photos.length, tapZone(event.nativeEvent.locationX, width)))
        }
        style={styles.photoTap}
      />

      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.bars}>
          {photos.map((photo, i) => (
            <View
              key={photo.id}
              style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.45)" }]}
            />
          ))}
        </View>
      ) : null}

      <LinearGradient colors={["transparent", "rgba(20,10,5,0.78)"]} pointerEvents="none" style={styles.gradient} />

      <Pressable
        accessibilityHint="Opens the full profile"
        accessibilityLabel={`${profile.displayName}, ${profile.age}${distance ? `, ${distance}` : ""}. View full profile`}
        accessibilityRole="button"
        onPress={onOpen}
        style={styles.info}
      >
        <View style={styles.nameRow}>
          <Text style={styles.white} variant="heading">
            {profile.displayName}, {profile.age}
          </Text>
          <Icon color="#FFFFFF" name="information-circle-outline" size={24} />
        </View>
        {distance || profile.city ? (
          <View style={styles.metaRow}>
            <Icon color="#FFFFFF" name="location-outline" size={16} />
            <Text style={styles.white} variant="small">
              {[profile.city, distance].filter(Boolean).join("  ·  ")}
            </Text>
          </View>
        ) : null}
        {profile.bio ? (
          <Text numberOfLines={2} style={styles.white} variant="small">
            {profile.bio}
          </Text>
        ) : null}
        {shownInterests.length > 0 ? (
          <View style={styles.chips}>
            {shownInterests.map((interest) => (
              <Chip key={interest.slug} label={interest.label} />
            ))}
            {profile.interests.length > shownInterests.length ? (
              <Chip label={`+${profile.interests.length - shownInterests.length}`} />
            ) : null}
          </View>
        ) : null}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  photoTap: { ...StyleSheet.absoluteFillObject, bottom: 150 },
  bars: { position: "absolute", top: spacing.xs, left: spacing.xs, right: spacing.xs, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, height: 260 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.md, gap: spacing.xs },
  nameRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  metaRow: { flexDirection: "row", alignItems: "center", gap: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.xxs, marginTop: -spacing.xxs },
  white: { color: "#FFFFFF" }
});
