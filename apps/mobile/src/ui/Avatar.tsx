import { Image } from "expo-image";
import React from "react";
import { View } from "react-native";
import { absoluteUrl } from "../api/client";
import { useTheme } from "../theme";
import { Text } from "./Text";

type Props = { uri?: string | null; name: string; size?: number };

export function Avatar({ uri, name, size = 48 }: Props) {
  const { colors } = useTheme();
  const radius = size / 2;
  if (uri) {
    return (
      <Image
        accessibilityLabel={`Photo de ${name}`}
        source={{ uri: absoluteUrl(uri) }}
        style={{ width: size, height: size, borderRadius: radius, backgroundColor: colors.surfaceMuted }}
        contentFit="cover"
        cachePolicy="memory-disk"
        transition={150}
      />
    );
  }
  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: radius,
        backgroundColor: colors.primarySoft,
        alignItems: "center",
        justifyContent: "center"
      }}
    >
      <Text variant="label" tone="primary">
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}
