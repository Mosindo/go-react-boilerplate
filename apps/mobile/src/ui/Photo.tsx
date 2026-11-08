import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { photoSource } from "../api/client";
import { useTheme } from "../theme/theme";
import { Text } from "./Text";

type PhotoProps = {
  path?: string;
  style?: StyleProp<ImageStyle>;
  contentFit?: "cover" | "contain";
  accessibilityLabel?: string;
};

/** Private photo loaded with the bearer token and cached on disk by its versioned URL. */
export function Photo({ path, style, contentFit = "cover", accessibilityLabel }: PhotoProps) {
  const { colors } = useTheme();
  if (!path) {
    return <View style={[{ backgroundColor: colors.surfaceMuted }, style as StyleProp<ViewStyle>]} />;
  }
  return (
    <Image
      source={photoSource(path)}
      style={style}
      contentFit={contentFit}
      transition={150}
      cachePolicy="disk"
      accessibilityLabel={accessibilityLabel}
      recyclingKey={path}
    />
  );
}

export function Avatar({ path, name, size = 48 }: { path?: string; name: string; size?: number }) {
  const { colors } = useTheme();
  const box = { width: size, height: size, borderRadius: size / 2 };
  if (path) {
    return <Photo path={path} style={[box, styles.clip]} accessibilityLabel={`Photo de ${name}`} />;
  }
  return (
    <View style={[box, styles.center, { backgroundColor: colors.primarySoft }]}>
      <Text variant="heading" tone="primary" style={{ fontSize: size * 0.4 }}>
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  clip: { overflow: "hidden" },
  center: { alignItems: "center", justifyContent: "center" }
});
