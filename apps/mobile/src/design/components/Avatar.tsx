import React from "react";
import { View } from "react-native";
import { Image } from "expo-image";
import { useTheme } from "../ThemeProvider";
import { mediaUrl } from "../../lib/api/config";
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
  return (
    <View
      accessibilityLabel={`Photo de ${name}`}
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        overflow: "hidden",
        backgroundColor: colors.primarySoft,
        alignItems: "center",
        justifyContent: "center",
        borderWidth: ring ? 2.5 : 0,
        borderColor: colors.primary
      }}
    >
      {uri ? (
        <Image cachePolicy="memory-disk" contentFit="cover" source={{ uri: mediaUrl(uri) }} style={{ width: "100%", height: "100%" }} transition={150} />
      ) : (
        <Text style={{ color: colors.primary, fontSize: size * 0.4, lineHeight: size * 0.5 }} variant="heading">
          {initial}
        </Text>
      )}
    </View>
  );
}
