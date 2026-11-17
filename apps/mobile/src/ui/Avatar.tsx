import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { colors } from "../theme";
import { Text } from "./Text";
import { mediaUrl } from "../api/client";

type AvatarProps = {
  url?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ url, name, size = 48, ring }: AvatarProps) {
  const dimension = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View style={[dimension, styles.wrap, ring ? styles.ring : null]}>
      {url ? (
        <Image
          source={{ uri: mediaUrl(url) }}
          style={dimension}
          contentFit="cover"
          cachePolicy="memory-disk"
          accessibilityLabel={`Photo de ${name}`}
          transition={150}
        />
      ) : (
        <Text variant="heading" tone="muted">
          {name.slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.surfaceRaised,
    overflow: "hidden"
  },
  ring: {
    borderWidth: 2,
    borderColor: colors.accent
  }
});
