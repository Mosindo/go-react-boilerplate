import { Ionicons } from "@expo/vector-icons";
import React from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  View,
  type PressableProps,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  size?: "md" | "lg";
  loading?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  style?: StyleProp<ViewStyle>;
};

export function Button({
  label,
  variant = "primary",
  size = "lg",
  loading,
  disabled,
  icon,
  style,
  ...props
}: ButtonProps) {
  const { colors, radii } = useTheme();
  const palette: Record<Variant, { bg: string; fg: string; border: string; pressed: string }> = {
    primary: { bg: colors.primary, fg: colors.onPrimary, border: colors.primary, pressed: colors.primaryPressed },
    secondary: { bg: colors.surface, fg: colors.text, border: colors.borderStrong, pressed: colors.surfaceAlt },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent", pressed: colors.primarySoft },
    danger: { bg: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft, pressed: colors.dangerSoft }
  };
  const p = palette[variant];
  const isDisabled = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      accessibilityLabel={label}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: size === "lg" ? 54 : 44,
          borderRadius: radii.pill,
          backgroundColor: pressed ? p.pressed : p.bg,
          borderColor: p.border,
          opacity: isDisabled ? 0.55 : 1
        },
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={p.fg} /> : icon ? <Ionicons name={icon} size={18} color={p.fg} /> : null}
        <Text variant="body" weight="semibold" style={{ color: p.fg }}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

type IconButtonProps = Omit<PressableProps, "style" | "children"> & {
  icon: keyof typeof Ionicons.glyphMap;
  accessibilityLabel: string;
  size?: number;
  tone?: "default" | "primary" | "muted" | "inverse";
  filled?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, size = 44, tone = "default", filled, style, ...props }: IconButtonProps) {
  const { colors } = useTheme();
  const fg = { default: colors.text, primary: colors.onPrimary, muted: colors.textMuted, inverse: "#FFFFFF" }[tone];
  const bg = filled ? (tone === "primary" ? colors.primary : colors.surface) : "transparent";
  return (
    <Pressable
      accessibilityRole="button"
      hitSlop={8}
      style={({ pressed }) => [
        styles.icon,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: bg,
          borderWidth: filled && tone !== "primary" ? StyleSheet.hairlineWidth : 0,
          borderColor: colors.border,
          opacity: pressed ? 0.7 : 1
        },
        style
      ]}
      {...props}
    >
      <Ionicons name={icon} size={size * 0.48} color={fg} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", paddingHorizontal: 22, borderWidth: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  icon: { alignItems: "center", justifyContent: "center" }
});
