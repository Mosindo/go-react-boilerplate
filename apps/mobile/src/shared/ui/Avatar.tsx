import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { PhotoImage } from "./PhotoImage";
import { Text } from "./Text";
import { controls } from "./tokens";

type AvatarProps = {
  photoId?: string | null;
  name?: string;
  size?: keyof typeof controls.avatar | number;
};

export function Avatar({ photoId, name, size = "md" }: AvatarProps) {
  const { colors } = useTheme();
  const px = typeof size === "number" ? size : controls.avatar[size];
  const shape = { width: px, height: px, borderRadius: px / 2 };

  if (!photoId) {
    return (
      <View accessibilityLabel={name} style={[styles.fallback, shape, { backgroundColor: colors.primarySoft }]}>
        <Text tone="primary" weight="bold">
          {name?.trim().charAt(0).toUpperCase() || "?"}
        </Text>
      </View>
    );
  }
  return <PhotoImage accessibilityLabel={name} photoId={photoId} size="thumb" style={shape} />;
}

const styles = StyleSheet.create({ fallback: { alignItems: "center", justifyContent: "center" } });
