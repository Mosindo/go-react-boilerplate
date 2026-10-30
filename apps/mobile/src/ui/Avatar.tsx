import React from "react";
import { Image, StyleSheet, View } from "react-native";
import { mediaUrl } from "../config";
import { Text } from "./Text";
import { useTheme } from "./theme";

type AvatarProps = { name: string; uri?: string; size?: number };

export function Avatar({ name, uri, size = 52 }: AvatarProps) {
  const { colors } = useTheme();
  const dimension = { width: size, height: size, borderRadius: size / 2 };

  if (uri) {
    return (
      <Image
        accessibilityIgnoresInvertColors
        source={{ uri: mediaUrl(uri) }}
        style={[dimension, { backgroundColor: colors.surfaceAlt }]}
      />
    );
  }
  return (
    <View style={[styles.fallback, dimension, { backgroundColor: colors.primarySoft }]}>
      <Text variant="heading" tone="primary">
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" }
});
