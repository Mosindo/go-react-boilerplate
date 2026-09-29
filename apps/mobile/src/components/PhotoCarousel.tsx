import React, { useCallback, useState } from "react";
import {
  FlatList,
  Image,
  StyleSheet,
  View,
  type LayoutChangeEvent,
  type NativeScrollEvent,
  type NativeSyntheticEvent
} from "react-native";
import type { Photo } from "../api/models";
import { useTheme } from "../shared/ui/theme";
import { Txt, photoTextColor } from "./kit";
import { photoUri } from "./photoUrl";

type Props = {
  photos: readonly Photo[];
  name: string;
  aspectRatio?: number;
  testID?: string;
};

/** Swipeable photo pager with dots. */
export function PhotoCarousel({ photos, name, aspectRatio = 1.2, testID }: Props) {
  const { colors } = useTheme();
  const [width, setWidth] = useState(0);
  const [index, setIndex] = useState(0);

  const onLayout = useCallback((e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width), []);
  const onScrollEnd = useCallback(
    (e: NativeSyntheticEvent<NativeScrollEvent>) => {
      if (width <= 0) return;
      setIndex(Math.round(e.nativeEvent.contentOffset.x / width));
    },
    [width]
  );

  const height = width > 0 ? width * aspectRatio : 320;

  return (
    <View
      onLayout={onLayout}
      testID={testID}
      style={{ height, backgroundColor: colors.surfaceMuted }}
    >
      {photos.length === 0 ? (
        <View style={styles.center}>
          <Txt variant="title" tone="muted">
            {name.charAt(0).toUpperCase()}
          </Txt>
        </View>
      ) : width > 0 ? (
        <FlatList
          data={photos}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          keyExtractor={(p) => p.id}
          onMomentumScrollEnd={onScrollEnd}
          getItemLayout={(_d, i) => ({
            length: width,
            offset: width * i,
            index: i
          })}
          renderItem={({ item, index: i }) => (
            <Image
              source={{ uri: photoUri(item) ?? undefined }}
              resizeMode="cover"
              accessibilityLabel={`${name}, photo ${i + 1} of ${photos.length}`}
              style={{ width, height }}
            />
          )}
        />
      ) : null}
      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.dot,
                {
                  backgroundColor: photoTextColor,
                  opacity: i === index ? 0.95 : 0.4
                }
              ]}
            />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  dots: {
    position: "absolute",
    bottom: 12,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6
  },
  dot: { width: 7, height: 7, borderRadius: 4 }
});
