import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { API_BASE_URL } from "../api/client";
import { useTheme } from "../theme";
import { Icon } from "./Icon";

/** Signed media URLs rotate hourly; caching by storage key keeps the disk cache warm across rotations. */
export function mediaSource(url: string) {
  const path = url.split("?")[0] ?? url;
  return { uri: url.startsWith("http") ? url : `${API_BASE_URL}${url}`, cacheKey: path };
}

export function Photo({
  url,
  style,
  accessibilityLabel,
  priority = "normal"
}: {
  url?: string;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
}) {
  const { colors } = useTheme();
  if (!url) {
    return (
      <View style={[styles.placeholder, { backgroundColor: colors.skeleton }, style as StyleProp<ViewStyle>]}>
        <Icon name="person" size={28} color={colors.textSubtle} />
      </View>
    );
  }
  return (
    <Image
      source={mediaSource(url)}
      style={[{ backgroundColor: colors.skeleton }, style]}
      contentFit="cover"
      transition={180}
      cachePolicy="memory-disk"
      priority={priority}
      accessibilityLabel={accessibilityLabel}
    />
  );
}

export function Avatar({ url, size = 52, label }: { url?: string; size?: number; label?: string }) {
  return <Photo url={url} accessibilityLabel={label} style={{ width: size, height: size, borderRadius: size / 2 }} />;
}

const styles = StyleSheet.create({
  placeholder: { alignItems: "center", justifyContent: "center" }
});
