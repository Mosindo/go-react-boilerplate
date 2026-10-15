import React, { useEffect, useState } from "react";
import { Pressable, View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import type { PhotoView } from "../api/types";
import { mediaUrl } from "../api/client";
import { sortedPhotos } from "../lib/photos";
import { useTheme } from "../theme";
import { spacing } from "../shared/ui";

type PhotoCarouselProps = {
  photos: PhotoView[];
  name?: string;
  style?: StyleProp<ViewStyle>;
  /** Tap the left/right half to browse. Disable on cards that are being dragged. */
  interactive?: boolean;
};

/**
 * One photo at a time with story-style progress bars. The next image is
 * prefetched so browsing feels instant without loading every photo up front.
 */
export function PhotoCarousel({ interactive = true, name, photos, style }: PhotoCarouselProps) {
  const palette = useTheme();
  const ordered = sortedPhotos(photos);
  const [index, setIndex] = useState(0);
  const current = ordered[Math.min(index, ordered.length - 1)];

  useEffect(() => {
    const next = ordered[index + 1];
    if (next) {
      void Image.prefetch(mediaUrl(next.url) ?? "", "memory-disk");
    }
  }, [index, ordered]);

  const go = (delta: number) => setIndex((i) => Math.max(0, Math.min(ordered.length - 1, i + delta)));

  return (
    <View style={[{ flex: 1, backgroundColor: palette.surfaceMuted, overflow: "hidden" }, style]}>
      {current ? (
        <Image
          accessibilityIgnoresInvertColors
          accessibilityLabel={name ? `Photo ${index + 1} sur ${ordered.length} de ${name}` : `Photo ${index + 1} sur ${ordered.length}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={current.id}
          source={{ uri: mediaUrl(current.url) }}
          style={{ ...fill }}
          transition={120}
        />
      ) : (
        <View style={{ ...fill, alignItems: "center", justifyContent: "center" }}>
          <Ionicons color={palette.textSubtle} name="person" size={72} />
        </View>
      )}

      {interactive && ordered.length > 1 ? (
        <View style={{ ...fill, flexDirection: "row" }}>
          <Pressable accessibilityLabel="Photo précédente" accessibilityRole="button" onPress={() => go(-1)} style={{ flex: 1 }} />
          <Pressable accessibilityLabel="Photo suivante" accessibilityRole="button" onPress={() => go(1)} style={{ flex: 1 }} />
        </View>
      ) : null}

      {ordered.length > 1 ? (
        <View
          pointerEvents="none"
          style={{ position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 }}
        >
          {ordered.map((photo, i) => (
            <View
              key={photo.id}
              style={{ flex: 1, height: 3.5, borderRadius: 2, backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }}
            />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const fill = { position: "absolute", top: 0, left: 0, right: 0, bottom: 0, width: "100%", height: "100%" } as const;
