import { Image, type ImageProps } from "expo-image";
import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { apiUrl } from "../api/client";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type PhotoProps = Omit<ImageProps, "source"> & { url: string | null | undefined };

/** Signed media URL rendered with memory + disk caching. */
export function Photo({ url, style, ...props }: PhotoProps) {
  const { colors } = useTheme();
  if (!url) {
    return <View style={[{ backgroundColor: colors.surfaceAlt }, style as StyleProp<ViewStyle>]} />;
  }
  return (
    <Image
      source={{ uri: apiUrl(url) }}
      cachePolicy="memory-disk"
      contentFit="cover"
      transition={180}
      style={[{ backgroundColor: colors.surfaceAlt }, style]}
      {...props}
    />
  );
}

type AvatarProps = { url?: string | null; name: string; size?: number };

export function Avatar({ url, name, size = 48 }: AvatarProps) {
  const { colors } = useTheme();
  const shape = { width: size, height: size, borderRadius: size / 2 };
  if (url) {
    return <Photo url={url} style={shape} accessibilityLabel={`Photo de ${name}`} />;
  }
  return (
    <View style={[shape, styles.fallback, { backgroundColor: colors.primarySoft }]}>
      <Text weight="semibold" tone="primary" style={{ fontSize: size * 0.4 }}>
        {name.slice(0, 1).toUpperCase() || "?"}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" }
});
