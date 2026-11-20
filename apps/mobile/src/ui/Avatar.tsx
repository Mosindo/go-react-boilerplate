import React from "react";
import { View } from "react-native";
import { Image } from "expo-image";
import { useTheme } from "../theme";
import { mediaUrl } from "../api/client";
import { Text } from "./Text";

type Props = { uri?: string | null; name: string; size?: number; ring?: boolean };

export function Avatar({ uri, name, size = 52, ring }: Props) {
  const { colors } = useTheme();
  const style = {
    width: size,
    height: size,
    borderRadius: size / 2,
    backgroundColor: colors.surfaceAlt,
    borderWidth: ring ? 2 : 0,
    borderColor: colors.primary
  };
  if (uri) {
    return (
      <Image
        accessibilityLabel={`Photo de ${name}`}
        cachePolicy="memory-disk"
        contentFit="cover"
        source={{ uri: mediaUrl(uri) }}
        style={style}
        transition={150}
      />
    );
  }
  return (
    <View style={[style, { alignItems: "center", justifyContent: "center" }]}>
      <Text tone="muted" variant="heading">
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}
