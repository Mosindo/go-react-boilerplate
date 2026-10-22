import React, { useState } from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent, type StyleProp, type ViewStyle } from "react-native";
import type { PhotoRef } from "../api/types";
import { colors, radii, spacing } from "../shared/ui";
import { AuthImage } from "./AuthImage";

type PhotoCarouselProps = {
  photos: PhotoRef[];
  name: string;
  style?: StyleProp<ViewStyle>;
  /** Tapping the middle of the photo (the sides change photo). */
  onPressCenter?: () => void;
  testID?: string;
};

/** One photo at a time with story-style progress bars; tap the left/right edges to navigate. */
export function PhotoCarousel({ photos, name, style, onPressCenter, testID }: PhotoCarouselProps) {
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];

  const handlePress = (locationX: number) => {
    if (photos.length > 1 && locationX < width * 0.3) {
      setIndex((value) => Math.max(0, value - 1));
    } else if (photos.length > 1 && locationX > width * 0.7) {
      setIndex((value) => Math.min(photos.length - 1, value + 1));
    } else {
      onPressCenter?.();
    }
  };

  return (
    <View onLayout={(event: LayoutChangeEvent) => setWidth(event.nativeEvent.layout.width)} style={[styles.frame, style]} testID={testID}>
      <AuthImage label={`Photo ${index + 1} de ${name}`} photo={current} placeholder={name.slice(0, 1)} style={StyleSheet.absoluteFill} />
      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.bars}>
          {photos.map((photo, i) => (
            <View key={photo.id} style={[styles.bar, i === index ? styles.barActive : null]} />
          ))}
        </View>
      ) : null}
      <Pressable
        accessibilityHint="Touchez les bords pour changer de photo"
        accessibilityLabel={`Photos de ${name}`}
        accessibilityRole="imagebutton"
        onPress={(event) => handlePress(event.nativeEvent.locationX)}
        style={StyleSheet.absoluteFill}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { borderRadius: radii.xl, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.45)" },
  barActive: { backgroundColor: "#ffffff" }
});
