import React, { useMemo } from "react";
import { Image, StyleSheet, View, type StyleProp, type TextStyle, type ViewStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { Text } from "./Text";
import { controls } from "./tokens";

export type AvatarProps = {
  name?: string;
  size?: number;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
  uri?: string | null;
};

export function getInitials(name?: string): string {
  const parts = (name ?? "").trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
}

const styles = StyleSheet.create({
  container: { alignItems: "center", justifyContent: "center", borderWidth: 1, overflow: "hidden" },
  image: { width: "100%", height: "100%" }
});

export function Avatar({ name, size = controls.avatar.md, style, textStyle, uri }: AvatarProps) {
  const { colors } = useTheme();
  const initials = useMemo(() => getInitials(name), [name]);

  return (
    <View
      style={[
        styles.container,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: colors.primarySoft,
          borderColor: colors.primaryBorder
        },
        style
      ]}
    >
      {uri ? (
        <Image source={{ uri }} style={styles.image} />
      ) : (
        <Text
          style={textStyle}
          tone="primary"
          variant={size >= controls.avatar.lg ? "heading" : "label"}
          weight="bold"
        >
          {initials}
        </Text>
      )}
    </View>
  );
}
