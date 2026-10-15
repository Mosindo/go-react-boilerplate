import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { controls } from "./tokens";

export type AvatarSize = keyof typeof controls.avatar;

export type AvatarProps = {
  name?: string;
  /** Absolute image URL. */
  uri?: string | null;
  size?: AvatarSize;
  ring?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function initialsOf(name: string | undefined): string {
  const trimmed = (name ?? "").trim();
  if (!trimmed) {
    return "?";
  }
  return Array.from(trimmed)[0].toUpperCase();
}

export function Avatar({ name, ring = false, size = "md", style, uri }: AvatarProps) {
  const palette = useTheme();
  const dimension = controls.avatar[size];

  return (
    <View
      accessibilityIgnoresInvertColors
      style={[
        {
          width: dimension,
          height: dimension,
          borderRadius: dimension / 2,
          backgroundColor: palette.brandSoft,
          alignItems: "center",
          justifyContent: "center",
          overflow: "hidden",
          borderWidth: ring ? 3 : 0,
          borderColor: palette.primary
        },
        style
      ]}
    >
      {uri ? (
        <Image
          accessibilityLabel={name ? `Photo de ${name}` : "Photo"}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={uri}
          source={{ uri }}
          style={{ width: "100%", height: "100%" }}
          transition={150}
        />
      ) : (
        <Text style={{ fontSize: dimension * 0.42 }} tone="brand" weight="bold">
          {initialsOf(name)}
        </Text>
      )}
    </View>
  );
}
