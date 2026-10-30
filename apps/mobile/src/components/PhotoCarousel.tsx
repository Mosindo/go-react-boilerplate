import React, { useState } from "react";
import { Image, Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import type { PhotoRef } from "../api/types";
import { mediaUrl } from "../config";
import { radii, spacing, useTheme } from "../ui";

type Props = {
  photos: PhotoRef[];
  /** Tap on the left/right half to move between photos (Discover card). */
  tapToBrowse?: boolean;
  rounded?: boolean;
};

export function PhotoCarousel({ photos, tapToBrowse = true, rounded = true }: Props) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const safeIndex = Math.min(index, Math.max(0, photos.length - 1));
  const photo = photos[safeIndex];

  const onLayout = (event: LayoutChangeEvent) => setWidth(event.nativeEvent.layout.width);
  const go = (delta: number) => setIndex((current) => Math.min(photos.length - 1, Math.max(0, current + delta)));

  return (
    <View
      onLayout={onLayout}
      style={[styles.wrap, rounded ? styles.rounded : null, { backgroundColor: colors.surfaceAlt }]}
    >
      {photo ? (
        <Image accessibilityIgnoresInvertColors source={{ uri: mediaUrl(photo.url) }} style={styles.image} />
      ) : null}
      {tapToBrowse && photos.length > 1 ? (
        <View style={StyleSheet.absoluteFill}>
          <View style={styles.tapRow}>
            <Pressable accessibilityLabel="Photo précédente" onPress={() => go(-1)} style={styles.tap} />
            <Pressable accessibilityLabel="Photo suivante" onPress={() => go(1)} style={styles.tap} />
          </View>
        </View>
      ) : null}
      {photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.bar,
                {
                  width: (width - spacing.md * 2 - spacing.xs * (photos.length - 1)) / photos.length,
                  backgroundColor: i === safeIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)"
                }
              ]}
            />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { width: "100%", height: "100%", overflow: "hidden" },
  rounded: { borderRadius: radii.xl },
  image: { width: "100%", height: "100%" },
  tapRow: { flex: 1, flexDirection: "row" },
  tap: { flex: 1 },
  bars: {
    position: "absolute",
    top: spacing.md,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: spacing.xs
  },
  bar: { height: 4, borderRadius: 2 }
});
