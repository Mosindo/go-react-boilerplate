import React, { useMemo } from "react";
import { Image, View, type StyleProp, type TextStyle, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";
import { controls } from "./tokens";

export type AvatarProps = {
  name?: string;
  size?: number;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
  uri?: string | null;
};

function getInitials(name?: string): string {
  const parts = (name ?? "").trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return (parts[0] ?? "").slice(0, 2).toUpperCase();
  }
  return `${parts[0]?.[0] ?? ""}${parts[1]?.[0] ?? ""}`.toUpperCase();
}

const makeStyles = (t: Theme) => ({
  container: {
    alignItems: "center" as const,
    justifyContent: "center" as const,
    backgroundColor: t.colors.primarySoft,
    borderWidth: 1,
    borderColor: t.colors.border,
    overflow: "hidden" as const
  },
  image: { width: "100%" as const, height: "100%" as const }
});

export function Avatar({ name, size = controls.avatar.md, style, textStyle, uri }: AvatarProps) {
  const styles = useThemedStyles(makeStyles);
  const initials = useMemo(() => getInitials(name), [name]);

  return (
    <View
      accessibilityLabel={name ? `${name}'s photo` : "Profile photo"}
      accessibilityRole="image"
      style={[styles.container, { width: size, height: size, borderRadius: size / 2 }, style]}
    >
      {uri ? (
        <Image accessibilityIgnoresInvertColors source={{ uri }} style={styles.image} />
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
