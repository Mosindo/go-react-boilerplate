import React, { useState } from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { resolveMediaUrl } from "../api/client";
import { useTheme } from "../theme";
import { Icon } from "../ui";

type Props = {
  photos: { id: string; url: string }[];
  name: string;
  style?: StyleProp<ViewStyle>;
  children?: React.ReactNode;
};

/** Tap the left/right half to browse photos, with story-style indicators. */
export function PhotoCarousel({ photos, name, style, children }: Props) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const current = photos[Math.min(index, photos.length - 1)];
  const count = photos.length;

  return (
    <View style={[styles.wrap, { backgroundColor: colors.surfaceAlt }, style]}>
      {current ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${count} de ${name}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={current.id}
          source={{ uri: resolveMediaUrl(current.url) }}
          style={StyleSheet.absoluteFill}
          transition={120}
        />
      ) : (
        <View style={[StyleSheet.absoluteFill, styles.placeholder]}>
          <Icon color="textSubtle" name="image-outline" size={48} />
        </View>
      )}
      {/* Prefetch the next photo so browsing feels instant. */}
      {photos[index + 1] ? <Image source={{ uri: resolveMediaUrl(photos[index + 1].url) }} style={styles.hidden} /> : null}
      {count > 1 ? (
        <>
          <View pointerEvents="none" style={styles.bars}>
            {photos.map((p, i) => (
              <View key={p.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
            ))}
          </View>
          <Pressable
            accessibilityLabel="Photo précédente"
            onPress={() => setIndex((i) => Math.max(0, i - 1))}
            style={[styles.zone, styles.left]}
          />
          <Pressable
            accessibilityLabel="Photo suivante"
            onPress={() => setIndex((i) => Math.min(count - 1, i + 1))}
            style={[styles.zone, styles.right]}
          />
        </>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { overflow: "hidden" },
  placeholder: { alignItems: "center", justifyContent: "center" },
  hidden: { width: 1, height: 1, opacity: 0, position: "absolute" },
  bars: { position: "absolute", top: 10, left: 10, right: 10, flexDirection: "row", gap: 4, zIndex: 2 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  zone: { position: "absolute", top: 0, bottom: "35%", width: "50%", zIndex: 1 },
  left: { left: 0 },
  right: { right: 0 }
});
