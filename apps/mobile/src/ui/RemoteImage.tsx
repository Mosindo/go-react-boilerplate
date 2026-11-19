import React from "react";
import { Image, type ImageStyle } from "expo-image";
import type { StyleProp } from "react-native";
import { mediaUrl } from "../api/config";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/** Cached image backed by expo-image (memory + disk cache, progressive transitions). */
export function RemoteImage({ path, style, accessibilityLabel, priority = "normal" }: Props) {
  const { colors } = useTheme();
  const uri = mediaUrl(path);
  return (
    <Image
      source={uri ? { uri, cacheKey: path?.split("?")[0] } : undefined}
      style={[{ backgroundColor: colors.surfaceMuted }, style]}
      contentFit="cover"
      transition={180}
      priority={priority}
      cachePolicy="memory-disk"
      accessibilityLabel={accessibilityLabel}
      accessible={!!accessibilityLabel}
    />
  );
}
