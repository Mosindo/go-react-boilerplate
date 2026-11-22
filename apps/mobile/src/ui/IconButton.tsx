import React from "react";
import { Pressable, StyleSheet, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, layout } from "./theme";

export type IconButtonProps = Omit<PressableProps, "style"> & {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  size?: number;
  color?: string;
  background?: string;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({
  icon,
  label,
  size = 44,
  color = colors.text,
  background = colors.surfaceRaised,
  style,
  ...props
}: IconButtonProps) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      hitSlop={layout.hitSlop}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: background },
        pressed ? styles.pressed : null,
        props.disabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <Ionicons color={color} name={icon} size={Math.round(size * 0.46)} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    alignItems: "center",
    justifyContent: "center"
  },
  pressed: {
    opacity: 0.8,
    transform: [{ scale: 0.94 }]
  },
  disabled: {
    opacity: 0.4
  }
});
