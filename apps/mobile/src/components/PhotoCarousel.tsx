import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Photo } from "../api/types";
import { PhotoImage, Text } from "../shared/ui";
import { radius, spacing, useTheme } from "../theme";

export type PhotoCarouselProps = {
  photos: Photo[];
  name: string;
  index: number;
  onIndexChange: (index: number) => void;
  /** Tapping the middle of the photo (e.g. open the full profile). Without it the middle is inert. */
  onCenterPress?: () => void;
};

/** Photo viewer: tap the left/right side to move, dots show the position. */
export function PhotoCarousel({ photos, name, index, onIndexChange, onCenterPress }: PhotoCarouselProps) {
  const theme = useTheme();
  const photo = photos[index];
  const previous = () => onIndexChange(Math.max(0, index - 1));
  const next = () => onIndexChange(Math.min(photos.length - 1, index + 1));
  return (
    <View style={[styles.root, { backgroundColor: theme.surfaceAlt }]}>
      {photo ? (
        <PhotoImage
          accessibilityLabel={`${name}, photo ${index + 1} of ${photos.length}`}
          photo={photo}
          style={StyleSheet.absoluteFill}
        />
      ) : (
        <View style={styles.noPhoto}>
          <Text tone="muted">No photo</Text>
        </View>
      )}
      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {photos.map((item, dot) => (
            <View
              key={item.id}
              style={[styles.dot, { backgroundColor: dot === index ? "#FFFFFF" : "rgba(255,255,255,0.45)" }]}
            />
          ))}
        </View>
      ) : null}
      <View style={styles.zones}>
        <Pressable
          accessibilityLabel="Previous photo"
          accessibilityRole="button"
          accessibilityState={{ disabled: index === 0 }}
          disabled={index === 0}
          onPress={previous}
          style={styles.side}
        />
        <Pressable
          accessibilityElementsHidden={!onCenterPress}
          accessibilityLabel={`View ${name}'s profile`}
          accessibilityRole="button"
          disabled={!onCenterPress}
          importantForAccessibility={onCenterPress ? "yes" : "no-hide-descendants"}
          onPress={onCenterPress}
          style={styles.center}
        />
        <Pressable
          accessibilityLabel="Next photo"
          accessibilityRole="button"
          accessibilityState={{ disabled: index >= photos.length - 1 }}
          disabled={index >= photos.length - 1}
          onPress={next}
          style={styles.side}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, overflow: "hidden", borderRadius: radius.lg },
  noPhoto: { flex: 1, alignItems: "center", justifyContent: "center" },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2 },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  side: { flex: 1 },
  center: { flex: 1 }
});
