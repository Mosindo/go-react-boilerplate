import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";

type IconButtonProps = {
  icon: IconName;
  label: string;
  onPress?: () => void;
  size?: number;
  variant?: "plain" | "surface" | "primary" | "outline";
  color?: string;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, variant = "plain", color, disabled, style, testID }: IconButtonProps) {
  const { colors } = useTheme();
  const bg =
    variant === "primary"
      ? colors.primary
      : variant === "surface"
        ? colors.surface
        : variant === "outline"
          ? colors.surface
          : "transparent";
  const fg = color ?? (variant === "primary" ? colors.onPrimary : colors.text);
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: bg,
          borderColor: variant === "outline" ? colors.border : "transparent",
          opacity: disabled ? 0.4 : pressed ? 0.75 : 1,
          transform: [{ scale: pressed ? 0.94 : 1 }]
        },
        variant === "surface" || variant === "outline" || variant === "primary"
          ? { shadowColor: colors.shadow, shadowOpacity: 0.12, shadowRadius: 12, shadowOffset: { width: 0, height: 4 }, elevation: 3 }
          : null,
        style
      ]}
      testID={testID}
    >
      <Icon color={fg} name={icon} size={Math.round(size * 0.48)} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", borderWidth: 1 }
});
