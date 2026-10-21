import { Image } from "expo-image";
import React, { useState } from "react";
import { type LayoutChangeEvent, type NativeScrollEvent, type NativeSyntheticEvent, ScrollView, StyleSheet, View } from "react-native";
import { absoluteUrl } from "../api/client";
import type { Photo } from "../api/types";
import { Avatar } from "../shared/ui";

type Props = { photos: Photo[]; name: string; aspectRatio?: number };

/** Horizontal, swipeable gallery with page dots. */
export function PhotoCarousel({ photos, name, aspectRatio = 3 / 4 }: Props) {
  const [width, setWidth] = useState(0);
  const [index, setIndex] = useState(0);

  const onLayout = (e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width);
  const onScroll = (e: NativeSyntheticEvent<NativeScrollEvent>) => {
    if (width > 0) setIndex(Math.round(e.nativeEvent.contentOffset.x / width));
  };

  if (photos.length === 0) {
    return (
      <View style={[styles.frame, { aspectRatio }, styles.center]}>
        <Avatar name={name} size={96} />
      </View>
    );
  }

  return (
    <View style={[styles.frame, { aspectRatio }]} onLayout={onLayout} testID="photo-carousel">
      {width > 0 ? (
        <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false} onScroll={onScroll} scrollEventThrottle={32}>
          {photos.map((photo, i) => (
            <Image
              key={photo.id}
              accessibilityLabel={`Photo ${i + 1} sur ${photos.length} de ${name}`}
              source={{ uri: absoluteUrl(photo.url) }}
              style={{ width, height: "100%" }}
              contentFit="cover"
              transition={150}
            />
          ))}
        </ScrollView>
      ) : null}
      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, i === index && styles.dotActive]} />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { width: "100%", backgroundColor: "#00000012", overflow: "hidden" },
  center: { alignItems: "center", justifyContent: "center" },
  dots: { position: "absolute", top: 10, left: 0, right: 0, flexDirection: "row", justifyContent: "center", gap: 5 },
  dot: { width: 22, height: 4, borderRadius: 2, backgroundColor: "#FFFFFF66" },
  dotActive: { backgroundColor: "#FFFFFF" }
});
