import React, { useState } from "react";
import { FlatList, StyleSheet, View, type LayoutChangeEvent } from "react-native";

import type { Photo } from "../api/types";
import { radii, useTheme } from "../theme";
import { PhotoImage } from "./PhotoImage";

export function PhotoCarousel({ photos, name }: { photos: Photo[]; name: string }) {
  const { colors } = useTheme();
  const [width, setWidth] = useState(0);
  const [index, setIndex] = useState(0);

  const onLayout = (e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width);

  if (photos.length <= 1) {
    return (
      <View style={styles.frame} onLayout={onLayout}>
        <PhotoImage photo={photos[0]} name={name} style={styles.fill} />
      </View>
    );
  }
  return (
    <View style={styles.frame} onLayout={onLayout}>
      {width > 0 ? (
        <FlatList
          data={photos}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          keyExtractor={(p) => p.id}
          getItemLayout={(_, i) => ({ length: width, offset: width * i, index: i })}
          onMomentumScrollEnd={(e) => setIndex(Math.round(e.nativeEvent.contentOffset.x / width))}
          renderItem={({ item }) => (
            <PhotoImage photo={item} name={name} style={{ width, height: "100%" }} />
          )}
        />
      ) : null}
      <View style={styles.dots} pointerEvents="none">
        {photos.map((p, i) => (
          <View
            key={p.id}
            style={[
              styles.dot,
              { backgroundColor: i === index ? colors.onPrimary : "rgba(255,255,255,0.5)" },
            ]}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { width: "100%", aspectRatio: 0.8, borderRadius: radii.lg, overflow: "hidden" },
  fill: { width: "100%", height: "100%" },
  dots: {
    position: "absolute",
    top: 10,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6,
  },
  dot: { width: 22, height: 4, borderRadius: 2 },
});
