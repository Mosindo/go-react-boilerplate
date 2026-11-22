import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { mediaUrl } from "../api/client";
import type { Photo } from "../api/types";
import { colors } from "../ui";

type Props = {
  photos: Photo[];
  name: string;
  rounded?: number;
};

/** Tap the right/left half to browse photos; segment bars show the position. */
export function PhotoCarousel({ photos, name, rounded = 0 }: Props) {
  const [index, setIndex] = useState(0);
  useEffect(() => setIndex(0), [photos]);
  const current = photos[index];

  return (
    <View style={[styles.wrap, { borderRadius: rounded }]}>
      {current ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${name}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={current.id}
          source={{ uri: mediaUrl(current.url) }}
          style={StyleSheet.absoluteFill}
          transition={120}
        />
      ) : null}
      {photos.length > 1 ? (
        <>
          <View pointerEvents="none" style={styles.bars}>
            {photos.map((p, i) => (
              <View key={p.id} style={[styles.bar, i === index ? styles.barOn : null]} />
            ))}
          </View>
          <View style={styles.zones}>
            <Pressable
              accessibilityLabel="Photo précédente"
              accessibilityRole="button"
              onPress={() => setIndex((i) => Math.max(0, i - 1))}
              style={styles.zone}
            />
            <Pressable
              accessibilityLabel="Photo suivante"
              accessibilityRole="button"
              onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
              style={styles.zone}
            />
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    ...StyleSheet.absoluteFillObject,
    overflow: "hidden",
    backgroundColor: colors.surfaceHigh
  },
  bars: {
    position: "absolute",
    top: 10,
    left: 10,
    right: 10,
    flexDirection: "row",
    gap: 4
  },
  bar: {
    flex: 1,
    height: 3,
    borderRadius: 2,
    backgroundColor: "rgba(255,255,255,0.35)"
  },
  barOn: {
    backgroundColor: "#FFFFFF"
  },
  zones: {
    ...StyleSheet.absoluteFillObject,
    flexDirection: "row",
    bottom: "35%"
  },
  zone: {
    flex: 1
  }
});
