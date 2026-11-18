import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { resolveMediaUrl } from "../api/client";
import { useTheme } from "../theme";
import { Text } from "./Text";

type Props = {
  uri?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ uri, name, size = 52, ring }: Props) {
  const { colors } = useTheme();
  const initial = name.trim().charAt(0).toUpperCase() || "?";
  const dimension = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View
      style={[
        styles.base,
        dimension,
        { backgroundColor: colors.surfaceAlt },
        ring ? { borderWidth: 2, borderColor: colors.accent, padding: 2 } : null
      ]}
    >
      {uri ? (
        <Image
          accessibilityIgnoresInvertColors
          accessibilityLabel={`Photo de ${name}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={uri}
          source={{ uri: resolveMediaUrl(uri) }}
          style={[styles.fill, { borderRadius: size / 2 }]}
          transition={150}
        />
      ) : (
        <Text color="textMuted" style={{ fontSize: size * 0.4 }} variant="heading">
          {initial}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", overflow: "hidden" },
  fill: { width: "100%", height: "100%" }
});
