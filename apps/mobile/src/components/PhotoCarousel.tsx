import { Ionicons } from "@expo/vector-icons";
import { Image } from "expo-image";
import React, { useState } from "react";
import { Pressable, StyleSheet, View, type LayoutChangeEvent } from "react-native";
import { absoluteUrl } from "../api/client";
import type { Photo } from "../api/types";
import { radii, useTheme } from "../theme";

type Props = {
  photos: Photo[];
  name: string;
  /** Use the small rendition (decks load many cards). */
  thumbnail?: boolean;
  borderRadius?: number;
};

/** Tap the left/right half to browse photos, with segment indicators on top. */
export function PhotoCarousel({ photos, name, thumbnail = false, borderRadius = radii.xl }: Props) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const [width, setWidth] = useState(0);
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];

  const onLayout = (e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width);
  const step = (dir: 1 | -1) => setIndex((i) => Math.min(photos.length - 1, Math.max(0, i + dir)));

  return (
    <View style={[styles.wrap, { borderRadius, backgroundColor: colors.surfaceMuted }]} onLayout={onLayout}>
      {current ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${name}`}
          source={{ uri: absoluteUrl(thumbnail ? current.url : current.url) }}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          cachePolicy="memory-disk"
          transition={120}
        />
      ) : (
        <View style={styles.placeholder}>
          <Ionicons name="person" size={72} color={colors.textMuted} />
        </View>
      )}
      {photos.length > 1 && width > 0 ? (
        <>
          <Pressable accessibilityLabel="Photo précédente" style={[styles.zone, { left: 0 }]} onPress={() => step(-1)} />
          <Pressable accessibilityLabel="Photo suivante" style={[styles.zone, { right: 0 }]} onPress={() => step(1)} />
          <View style={styles.indicators} pointerEvents="none">
            {photos.map((p, i) => (
              <View key={p.id} style={[styles.segment, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
            ))}
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, overflow: "hidden" },
  placeholder: { flex: 1, alignItems: "center", justifyContent: "center" },
  zone: { position: "absolute", top: 0, bottom: 0, width: "50%" },
  indicators: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  segment: { flex: 1, height: 4, borderRadius: 2 }
});
