import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { absoluteUrl } from "../api/config";
import type { Photo as PhotoType } from "../api/types";
import { useTheme } from "../theme/ThemeProvider";
import { Icon } from "./Icon";

type Props = {
  photo?: PhotoType | null;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/**
 * Cached remote photo. The cache key is the photo id (signed URLs rotate
 * hourly), so the same picture is downloaded once per device.
 */
export function RemotePhoto({ photo, style, accessibilityLabel, priority = "normal" }: Props) {
  const { colors } = useTheme();
  if (!photo) {
    return (
      <View style={[style as StyleProp<ViewStyle>, styles.placeholder, { backgroundColor: colors.surfaceMuted }]}>
        <Icon color={colors.textSubtle} name="person-outline" size={28} />
      </View>
    );
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      cachePolicy="memory-disk"
      contentFit="cover"
      priority={priority}
      recyclingKey={photo.id}
      source={{ uri: absoluteUrl(photo.url), cacheKey: `photo-${photo.id}` }}
      style={[{ backgroundColor: colors.surfaceMuted }, style]}
      transition={180}
    />
  );
}

export function Avatar({ photo, size = 52, label }: { photo?: PhotoType | null; size?: number; label?: string }) {
  return <RemotePhoto accessibilityLabel={label} photo={photo} style={{ width: size, height: size, borderRadius: size / 2 }} />;
}

const styles = StyleSheet.create({
  placeholder: { alignItems: "center", justifyContent: "center" }
});
