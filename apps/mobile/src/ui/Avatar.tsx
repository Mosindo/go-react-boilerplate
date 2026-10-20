import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

export function Avatar({ url, name, size = 52 }: { url: string | null; name: string; size?: number }) {
  const { colors } = useTheme();
  const box = { width: size, height: size, borderRadius: size / 2 };
  if (url) {
    return <RemoteImage url={url} style={[styles.image, box]} accessibilityLabel={`Photo of ${name}`} />;
  }
  return (
    <View style={[styles.image, box, styles.fallback, { backgroundColor: colors.surfaceAlt }]}>
      <Text variant="label" tone="primary">
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  image: { overflow: "hidden" },
  fallback: { alignItems: "center", justifyContent: "center" }
});
