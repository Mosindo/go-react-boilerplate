import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/palette";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger" | "inverse";

type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: ButtonVariant;
  size?: "sm" | "md" | "lg";
  icon?: IconName;
  loading?: boolean;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", size = "md", icon, loading, fullWidth, disabled, style, ...props }: ButtonProps) {
  const { colors } = useTheme();
  const isDisabled = disabled || loading;

  const palette: Record<ButtonVariant, { bg: string; fg: string; border: string; pressed: string }> = {
    primary: { bg: colors.primary, fg: colors.onPrimary, border: colors.primary, pressed: colors.primaryPressed },
    secondary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primarySoft, pressed: colors.primarySoft },
    outline: { bg: "transparent", fg: colors.text, border: colors.borderStrong, pressed: colors.surfaceMuted },
    ghost: { bg: "transparent", fg: colors.textMuted, border: "transparent", pressed: colors.surfaceMuted },
    danger: { bg: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft, pressed: colors.dangerSoft },
    // For use on dark scrims/photos, independent of the colour scheme.
    inverse: { bg: "transparent", fg: "#FFFFFF", border: "rgba(255,255,255,0.55)", pressed: "rgba(255,255,255,0.12)" }
  };
  const p = palette[variant];
  const height = size === "lg" ? 56 : size === "sm" ? 38 : 48;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: height,
          paddingHorizontal: size === "sm" ? spacing.md : spacing.xl,
          backgroundColor: pressed ? p.pressed : p.bg,
          borderColor: p.border,
          opacity: isDisabled ? 0.5 : 1,
          transform: [{ scale: pressed ? 0.98 : 1 }]
        },
        fullWidth ? styles.full : null,
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={p.fg} size="small" /> : icon ? <Icon name={icon} size={18} color={p.fg} /> : null}
        <Text variant={size === "sm" ? "label" : "bodyStrong"} style={{ color: p.fg }}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

type IconButtonProps = Omit<PressableProps, "style" | "children"> & {
  icon: IconName;
  accessibilityLabel: string;
  variant?: "surface" | "primary" | "ghost" | "pass";
  size?: number;
  iconSize?: number;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, variant = "ghost", size = 44, iconSize, style, disabled, ...props }: IconButtonProps) {
  const { colors } = useTheme();
  const bg = { surface: colors.surface, primary: colors.primary, ghost: "transparent", pass: colors.surface }[variant];
  const fg = { surface: colors.text, primary: colors.onPrimary, ghost: colors.text, pass: colors.pass }[variant];
  return (
    <Pressable
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      style={({ pressed }) => [
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: bg,
          borderWidth: variant === "surface" || variant === "pass" ? StyleSheet.hairlineWidth : 0,
          borderColor: colors.border,
          opacity: disabled ? 0.4 : pressed ? 0.75 : 1,
          transform: [{ scale: pressed ? 0.94 : 1 }]
        },
        style
      ]}
      {...props}
    >
      <Icon name={icon} size={iconSize ?? Math.round(size * 0.48)} color={fg} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  full: { alignSelf: "stretch" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs }
});
