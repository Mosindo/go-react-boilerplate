import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { mediaUrl } from "../api/client";
import { Text } from "./Text";
import { colors } from "./theme";

type AvatarProps = {
  url?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ url, name, size = 52, ring }: AvatarProps) {
  const dimension = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View style={[styles.wrap, dimension, ring ? styles.ring : null]}>
      {url ? (
        <Image
          accessibilityIgnoresInvertColors
          accessibilityLabel={`Photo de ${name}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={url}
          source={{ uri: mediaUrl(url) }}
          style={dimension}
          transition={150}
        />
      ) : (
        <Text style={{ fontSize: size * 0.4 }} tone="accent" variant="heading">
          {name.slice(0, 1).toUpperCase() || "?"}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.surfaceHigh,
    overflow: "hidden"
  },
  ring: {
    borderWidth: 2,
    borderColor: colors.accent
  }
});
