import React, { useState } from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import type { PhotoRef } from "../api/types";
import { PhotoImage } from "../shared/media/PhotoImage";
import { radii, spacing } from "../shared/ui";

type PhotoCarouselProps = {
  photos: PhotoRef[];
  name: string;
  style?: StyleProp<ViewStyle>;
};

/** Tap the right/left half to move between photos; dots show the position. */
export function PhotoCarousel({ photos, name, style }: PhotoCarouselProps) {
  const [index, setIndex] = useState(0);
  const count = photos.length;
  const current = photos[Math.min(index, Math.max(count - 1, 0))];

  const step = (delta: number) => setIndex((i) => Math.max(0, Math.min(count - 1, i + delta)));

  return (
    <View style={[styles.root, style]}>
      <PhotoImage
        accessibilityLabel={`Photo ${index + 1} sur ${Math.max(count, 1)} de ${name}`}
        fallbackLabel={name}
        path={current?.url}
        style={StyleSheet.absoluteFillObject}
      />
      {count > 1 ? (
        <>
          <View pointerEvents="none" style={styles.dots}>
            {photos.map((photo, i) => (
              <View key={photo.id} style={[styles.dot, i === index ? styles.dotActive : styles.dotIdle]} />
            ))}
          </View>
          <View style={styles.zones}>
            <Pressable accessibilityLabel="Photo précédente" accessibilityRole="button" onPress={() => step(-1)} style={styles.zone} />
            <Pressable accessibilityLabel="Photo suivante" accessibilityRole="button" onPress={() => step(1)} style={styles.zone} />
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { overflow: "hidden", borderRadius: radii.xl },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  dotActive: { backgroundColor: "rgba(255,255,255,0.95)" },
  dotIdle: { backgroundColor: "rgba(255,255,255,0.35)" },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 }
});
