import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  loading?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  size?: "md" | "lg";
  style?: ViewStyle;
};

export function Button({
  label,
  variant = "primary",
  loading,
  icon,
  size = "lg",
  disabled,
  style,
  ...rest
}: ButtonProps) {
  const { colors, radii } = useTheme();
  const isDisabled = disabled || loading;
  const palette = {
    primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.surface, pressed: colors.surfaceSunken, fg: colors.text, border: colors.borderStrong },
    ghost: { bg: "transparent", pressed: colors.surfaceSunken, fg: colors.textMuted, border: "transparent" },
    danger: { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft }
  }[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: size === "lg" ? 54 : 44,
          borderRadius: radii.pill,
          backgroundColor: pressed ? palette.pressed : palette.bg,
          borderColor: palette.border,
          opacity: isDisabled ? 0.55 : 1,
          transform: [{ scale: pressed ? 0.985 : 1 }]
        },
        style
      ]}
      {...rest}
    >
      <View style={styles.row}>
        {loading ? (
          <ActivityIndicator color={palette.fg} />
        ) : icon ? (
          <Ionicons name={icon} size={18} color={palette.fg} />
        ) : null}
        <Text variant="label" style={{ color: palette.fg, fontSize: size === "lg" ? 16 : 15 }}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { borderWidth: 1, paddingHorizontal: 22, alignItems: "center", justifyContent: "center" },
  row: { flexDirection: "row", alignItems: "center", gap: 8 }
});
