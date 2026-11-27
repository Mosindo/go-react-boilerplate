import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { absoluteUrl } from "../api/client";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type RemoteImageProps = {
  url?: string;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/** Cached remote image (signed API URLs resolve against the API base). */
export function RemoteImage({ url, style, accessibilityLabel, priority = "normal" }: RemoteImageProps) {
  const { colors } = useTheme();
  return (
    <Image
      source={url ? { uri: absoluteUrl(url) } : undefined}
      style={[{ backgroundColor: colors.surfaceMuted }, style]}
      contentFit="cover"
      transition={180}
      cachePolicy="memory-disk"
      priority={priority}
      accessibilityLabel={accessibilityLabel}
      recyclingKey={url}
    />
  );
}

export function prefetchImages(urls: string[]) {
  if (urls.length > 0) {
    void Image.prefetch(urls.map(absoluteUrl), "memory-disk");
  }
}

type AvatarProps = {
  url?: string;
  name: string;
  size?: number;
  style?: StyleProp<ViewStyle>;
  ring?: boolean;
};

export function Avatar({ url, name, size = 48, style, ring }: AvatarProps) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        { width: size, height: size, borderRadius: size / 2, overflow: "hidden", backgroundColor: colors.primarySoft },
        ring ? { borderWidth: 2, borderColor: colors.primary } : null,
        styles.center,
        style
      ]}
    >
      {url ? (
        <RemoteImage url={url} style={{ width: size, height: size }} accessibilityLabel={`Photo de ${name}`} />
      ) : (
        <Text variant="heading" tone="primary">
          {name.slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View style={[styles.badge, { backgroundColor: colors.primary }]} accessibilityLabel={`${count} non lus`}>
      <Text variant="overline" style={{ color: colors.onPrimary, letterSpacing: 0 }}>
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  center: { alignItems: "center", justifyContent: "center" },
  badge: { minWidth: 22, height: 22, borderRadius: 11, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" }
});
