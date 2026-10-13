import React, { useState, type ReactNode } from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import type { PhotoRef } from "../../api/types";
import { PhotoImage, spacing } from "../../shared/ui";

// Stacked translucent bands approximate a bottom gradient without a gradient dependency.
const SHADE_STEPS = Array.from({ length: 28 }, (_, i) => Number((0.62 * (i / 27) ** 1.6).toFixed(3)));

type PhotoStackProps = {
  photos: PhotoRef[];
  name: string;
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

/**
 * A profile's photos shown one at a time. Tapping the left or right third moves
 * between photos; bars at the top show the position.
 */
export function PhotoStack({ photos, name, children, style }: PhotoStackProps) {
  const [index, setIndex] = useState(0);
  const current = photos[Math.min(index, photos.length - 1)];
  const go = (delta: number) => setIndex((i) => Math.max(0, Math.min(photos.length - 1, i + delta)));

  return (
    <View style={[styles.root, style]}>
      <PhotoImage accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${name}`} photoId={current?.id} style={styles.image} />
      <View pointerEvents="none" style={styles.shade}>
        {SHADE_STEPS.map((opacity, i) => (
          <View key={i} style={{ flex: 1, backgroundColor: `rgba(20, 8, 4, ${opacity})` }} />
        ))}
      </View>
      {photos.length > 1 ? (
        <>
          <View pointerEvents="none" style={styles.bars}>
            {photos.map((photo, i) => (
              <View key={photo.id} style={[styles.bar, { opacity: i === index ? 1 : 0.4 }]} />
            ))}
          </View>
          <View style={styles.zones}>
            <Pressable accessibilityLabel="Photo précédente" onPress={() => go(-1)} style={styles.zone} />
            <View style={styles.zoneSpacer} />
            <Pressable accessibilityLabel="Photo suivante" onPress={() => go(1)} style={styles.zone} testID="photo-next" />
          </View>
        </>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { overflow: "hidden", backgroundColor: "#000" },
  image: { ...StyleSheet.absoluteFillObject, width: "100%", height: "100%" },
  shade: { position: "absolute", left: 0, right: 0, bottom: 0, height: "55%" },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: spacing.xxs },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#FFFFFF" },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 },
  zoneSpacer: { flex: 1 }
});
