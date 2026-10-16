import React, { useEffect, useState } from "react";
import { Image, Pressable, StyleSheet, View } from "react-native";
import { absoluteUrl } from "../api/client";
import type { PhotoRef } from "../api/types";
import { Text, colors, radii, spacing } from "../shared/ui";

type PhotoCarouselProps = {
  photos: PhotoRef[];
  name: string;
  /** Tapping the left/right third flips photos; the middle calls onCenterPress. */
  onCenterPress?: () => void;
  rounded?: boolean;
};

export function PhotoCarousel({ photos, name, onCenterPress, rounded = true }: PhotoCarouselProps) {
  const [index, setIndex] = useState(0);
  const count = photos.length;

  const firstPhotoId = photos[0]?.id;
  useEffect(() => {
    setIndex(0);
  }, [firstPhotoId]);

  const go = (delta: number) => setIndex((current) => Math.min(Math.max(current + delta, 0), Math.max(count - 1, 0)));
  const current = photos[index];

  return (
    <View style={[styles.frame, rounded ? styles.rounded : null]}>
      {current ? (
        <Image
          accessibilityIgnoresInvertColors
          accessibilityLabel={`Photo ${index + 1} sur ${count} de ${name}`}
          resizeMode="cover"
          source={{ uri: absoluteUrl(current.url) }}
          style={styles.image}
        />
      ) : (
        <View style={styles.placeholder}>
          <Text tone="muted">Pas de photo</Text>
        </View>
      )}

      {count > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.dot, i === index ? styles.dotActive : null]} />
          ))}
        </View>
      ) : null}

      <View style={styles.zones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => go(-1)} style={styles.zone} />
        <Pressable accessibilityLabel="Voir le profil" onPress={onCenterPress} style={styles.zoneCenter} />
        <Pressable accessibilityLabel="Photo suivante" onPress={() => go(1)} style={styles.zone} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { flex: 1, backgroundColor: colors.surfaceSubtle, overflow: "hidden" },
  rounded: { borderRadius: radii.xl },
  image: { ...StyleSheet.absoluteFillObject, width: "100%", height: "100%" },
  placeholder: { flex: 1, alignItems: "center", justifyContent: "center" },
  dots: { position: "absolute", top: spacing.md, left: spacing.lg, right: spacing.lg, flexDirection: "row", gap: spacing.xs },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.45)" },
  dotActive: { backgroundColor: "#ffffff" },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 },
  zoneCenter: { flex: 1 }
});
