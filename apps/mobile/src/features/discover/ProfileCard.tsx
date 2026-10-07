import React from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import { Text, radii, spacing } from "../../shared/ui";
import type { PublicProfile } from "../types";
import { Chip } from "../ui/Chip";
import { PhotoImage } from "../ui/PhotoImage";
import { usePalette } from "../ui/palette";
import { formatDistance } from "./distance";
import { tapStep } from "./swipe";

type Props = {
  profile: PublicProfile;
  photoIndex: number;
  onPhotoStep: (step: -1 | 1) => void;
  onInfo: () => void;
  children?: React.ReactNode;
};

export function ProfileCard({ profile, photoIndex, onPhotoStep, onInfo, children }: Props) {
  const p = usePalette();
  const widthRef = React.useRef(300);
  const photo = profile.photos[photoIndex] ?? profile.photos[0] ?? null;
  const distance = formatDistance(profile.distanceKm);
  const shared = new Set(profile.sharedInterests);
  const ordered = [...profile.interests]
    .sort((a, b) => Number(shared.has(b.slug)) - Number(shared.has(a.slug)))
    .slice(0, 6);

  const onLayout = (e: LayoutChangeEvent) => {
    widthRef.current = e.nativeEvent.layout.width;
  };

  return (
    <View
      onLayout={onLayout}
      style={[styles.card, { backgroundColor: p.surfaceMuted }]}
      testID="discover-card"
    >
      {photo ? (
        <PhotoImage
          accessibilityLabel={`Photo ${photoIndex + 1} of ${profile.photos.length} of ${profile.firstName}`}
          photo={photo}
          style={StyleSheet.absoluteFill}
        />
      ) : (
        <View style={[StyleSheet.absoluteFill, styles.noPhoto]}>
          <Text tone="muted">No photo yet</Text>
        </View>
      )}

      <Pressable
        accessibilityHint="Tap the left side for the previous photo and the right side for the next one"
        accessibilityLabel={`${profile.firstName}'s photos`}
        onPress={(e) => onPhotoStep(tapStep(e.nativeEvent.locationX, widthRef.current))}
        style={StyleSheet.absoluteFill}
        testID="discover-photo-area"
      />

      {profile.photos.length > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {profile.photos.map((ph, i) => (
            <View
              key={ph.id}
              style={[
                styles.dot,
                { backgroundColor: i === photoIndex ? "#ffffff" : "rgba(255,255,255,0.4)" }
              ]}
            />
          ))}
        </View>
      ) : null}

      <View pointerEvents="box-none" style={styles.bottom}>
        <View pointerEvents="none" style={styles.scrim} />
        <View style={styles.infoRow}>
          <View pointerEvents="none" style={styles.infoText}>
            <Text numberOfLines={1} style={styles.name} weight="bold">
              {profile.firstName}, {profile.age}
            </Text>
            {profile.city ? (
              <Text numberOfLines={1} style={styles.meta}>
                {profile.city}
              </Text>
            ) : null}
            {distance ? <Text style={styles.meta}>{distance}</Text> : null}
          </View>
          <Pressable
            accessibilityLabel={`View ${profile.firstName}'s full profile`}
            accessibilityRole="button"
            hitSlop={8}
            onPress={onInfo}
            style={styles.info}
            testID="discover-info-button"
          >
            <Text style={styles.infoGlyph} weight="bold">
              i
            </Text>
          </Pressable>
        </View>
        {ordered.length > 0 ? (
          <View pointerEvents="none" style={styles.chips}>
            {ordered.map((interest) => (
              <Chip
                highlighted={shared.has(interest.slug)}
                key={interest.slug}
                label={interest.label}
                onDark
              />
            ))}
          </View>
        ) : null}
        {profile.bio ? (
          <Text numberOfLines={2} pointerEvents="none" style={styles.bio}>
            {profile.bio}
          </Text>
        ) : null}
      </View>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden" },
  noPhoto: { alignItems: "center", justifyContent: "center" },
  dots: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: 4
  },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  bottom: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, gap: spacing.sm },
  scrim: { ...StyleSheet.absoluteFillObject, backgroundColor: "rgba(0,0,0,0.42)" },
  infoRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.md },
  infoText: { flex: 1 },
  name: { color: "#ffffff", fontSize: 28, lineHeight: 34 },
  meta: { color: "rgba(255,255,255,0.9)", fontSize: 14, lineHeight: 20 },
  info: {
    width: 36,
    height: 36,
    borderRadius: 18,
    borderWidth: 1.5,
    borderColor: "#ffffff",
    alignItems: "center",
    justifyContent: "center"
  },
  infoGlyph: { color: "#ffffff", fontSize: 16, lineHeight: 20 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  bio: { color: "#ffffff", fontSize: 14, lineHeight: 20 }
});
