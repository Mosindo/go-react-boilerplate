import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { colors } from "../theme/tokens";
import { Text } from "./Text";
import { mediaUrl } from "../api/client";

export type AvatarProps = {
  uri?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ uri, name, size = 48, ring }: AvatarProps) {
  const radius = size / 2;
  return (
    <View
      accessibilityLabel={`Photo de ${name}`}
      style={[styles.base, { width: size, height: size, borderRadius: radius }, ring && { borderWidth: 2, borderColor: colors.accent, padding: 2 }]}
    >
      {uri ? (
        <Image source={{ uri: mediaUrl(uri) }} style={{ width: "100%", height: "100%", borderRadius: radius }} contentFit="cover" transition={150} />
      ) : (
        <Text variant="bodyStrong" tone="muted">
          {name.slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  base: { backgroundColor: colors.surfaceRaised, alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
