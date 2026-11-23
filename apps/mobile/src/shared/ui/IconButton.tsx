import React from "react";
import { Pressable, StyleSheet, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, controls } from "./tokens";

type IconButtonProps = Omit<PressableProps, "style" | "children"> & {
  glyph: string;
  accessibilityLabel: string;
  size?: number;
  tone?: "default" | "primary" | "danger" | "muted";
  filled?: boolean;
  style?: StyleProp<ViewStyle>;
};

const toneColor = {
  default: colors.text,
  primary: colors.primary,
  danger: colors.danger,
  muted: colors.textMuted
};

/** Round button rendering a text glyph (keeps the app free of icon-font dependencies). */
export function IconButton({ filled = false, glyph, size = 44, style, tone = "default", ...props }: IconButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      hitSlop={controls.hitSlop}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2 },
        filled ? { backgroundColor: tone === "primary" ? colors.primary : colors.surfaceRaised, borderColor: "transparent" } : null,
        pressed ? styles.pressed : null,
        props.disabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <Text
        style={{ fontSize: size * 0.42, lineHeight: size * 0.52, color: filled && tone === "primary" ? colors.onPrimary : toneColor[tone] }}
        weight="bold"
      >
        {glyph}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  pressed: {
    transform: [{ scale: 0.94 }],
    opacity: 0.85
  },
  disabled: {
    opacity: 0.4
  }
});
