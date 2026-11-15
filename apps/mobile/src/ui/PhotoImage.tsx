import React from "react";
import { StyleSheet, View, type ImageStyle, type StyleProp } from "react-native";
import { Image } from "expo-image";
import { absoluteUrl } from "../api/client";
import type { Photo } from "../api/types";
import { useTheme } from "../theme";

/**
 * Signed media URLs rotate every few hours; the cache key is the stable
 * object path so images are fetched once and reused.
 */
export function PhotoImage({
  photo,
  style,
  accessibilityLabel,
  priority = "normal"
}: {
  photo: Photo | undefined;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
}) {
  const { colors } = useTheme();
  if (!photo) {
    return <View style={[styles.placeholder, { backgroundColor: colors.surfaceAlt }, style as object]} />;
  }
  const cacheKey = photo.url.split("?")[0];
  return (
    <Image
      source={{ uri: absoluteUrl(photo.url), cacheKey }}
      style={[{ backgroundColor: colors.surfaceAlt }, style]}
      contentFit="cover"
      transition={180}
      cachePolicy="memory-disk"
      priority={priority}
      recyclingKey={cacheKey}
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
    />
  );
}

const styles = StyleSheet.create({ placeholder: {} });
