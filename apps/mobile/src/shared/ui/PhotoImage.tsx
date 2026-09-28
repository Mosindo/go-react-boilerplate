import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { photoSource } from "../../api/photos";
import type { Photo } from "../../api/types";
import { useAccessToken } from "../../hooks/useAccessToken";
import { useTheme } from "../../theme";

export type PhotoImageProps = {
  photo: Photo;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  contentFit?: "cover" | "contain";
};

/** Renders a protected photo (bearer header + token independent cache key). */
export function PhotoImage({ photo, style, accessibilityLabel, contentFit = "cover" }: PhotoImageProps) {
  const token = useAccessToken();
  const theme = useTheme();
  if (!token) {
    return <View style={[{ backgroundColor: theme.skeleton }, style as StyleProp<ViewStyle>]} />;
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
      cachePolicy="disk"
      contentFit={contentFit}
      source={photoSource(photo, token)}
      style={style}
      transition={150}
    />
  );
}
