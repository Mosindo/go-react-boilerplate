import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { mediaUrl } from "../api/client";
import type { Photo } from "../api/types";
import { colors, radii } from "../theme/tokens";

type Props = {
  photos: Photo[];
  name: string;
  rounded?: boolean;
  children?: React.ReactNode;
};

/** Tap the left/right half to browse photos. Progress bars show position. */
export function PhotoCarousel({ photos, name, rounded = true, children }: Props) {
  const [index, setIndex] = useState(0);
  const count = photos.length;
  const current = photos[Math.min(index, count - 1)];

  return (
    <View style={[styles.root, rounded && styles.rounded]}>
      {current ? (
        <Image
          source={{ uri: mediaUrl(current.url) }}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          transition={120}
          cachePolicy="memory-disk"
          accessibilityLabel={`Photo ${index + 1} sur ${count} de ${name}`}
        />
      ) : (
        <View style={[StyleSheet.absoluteFill, styles.placeholder]} />
      )}
      {photos.slice(index + 1, index + 2).map((p) => (
        // Prefetch the next photo so browsing feels instant.
        <Image key={p.id} source={{ uri: mediaUrl(p.url) }} style={styles.prefetch} cachePolicy="memory-disk" />
      ))}
      {count > 1 ? (
        <>
          <View style={styles.bars} pointerEvents="none">
            {photos.map((p, i) => (
              <View key={p.id} style={[styles.bar, i === index && styles.barActive]} />
            ))}
          </View>
          <Pressable
            style={[styles.zone, { left: 0 }]}
            onPress={() => setIndex((i) => Math.max(0, i - 1))}
            accessibilityRole="button"
            accessibilityLabel="Photo précédente"
          />
          <Pressable
            style={[styles.zone, { right: 0 }]}
            onPress={() => setIndex((i) => Math.min(count - 1, i + 1))}
            accessibilityRole="button"
            accessibilityLabel="Photo suivante"
          />
        </>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, overflow: "hidden", backgroundColor: colors.surfaceRaised },
  rounded: { borderRadius: radii.xl },
  placeholder: { backgroundColor: colors.surfaceRaised },
  prefetch: { width: 1, height: 1, opacity: 0, position: "absolute" },
  bars: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4, zIndex: 2 },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.35)" },
  barActive: { backgroundColor: colors.white },
  zone: { position: "absolute", top: 0, bottom: "35%", width: "50%", zIndex: 1 }
});
