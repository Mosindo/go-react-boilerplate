import { Image } from "expo-image";
import React from "react";
import type { StyleProp, ImageStyle } from "react-native";
import { authImageSource } from "../api/client";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  url: string;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  contentFit?: "cover" | "contain";
};

/** Photos are private: every request carries the access token. */
export function AuthImage({ url, style, accessibilityLabel, contentFit = "cover" }: Props) {
  const theme = useTheme();
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      cachePolicy="disk"
      contentFit={contentFit}
      recyclingKey={url}
      source={authImageSource(url)}
      style={[{ backgroundColor: theme.surfaceMuted }, style]}
      transition={150}
    />
  );
}
