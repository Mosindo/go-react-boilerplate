import React from "react";
import { Image } from "expo-image";
import { type StyleProp, type ImageStyle } from "react-native";
import { assetUrl } from "../../api/client";
import { useColors } from "./theme";

type Props = { path: string | null | undefined; style?: StyleProp<ImageStyle>; accessibilityLabel?: string; priority?: "low" | "normal" | "high" };

/** Image served by the API through a signed relative URL, cached on disk and in memory. */
export function RemoteImage({ path, style, accessibilityLabel, priority = "normal" }: Props) {
  const c = useColors();
  return (
    <Image
      source={path ? { uri: assetUrl(path) } : undefined}
      style={[{ backgroundColor: c.surfaceMuted }, style]}
      contentFit="cover"
      transition={150}
      cachePolicy="memory-disk"
      priority={priority}
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
    />
  );
}
