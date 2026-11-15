import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type ViewStyle } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  loading?: boolean;
  icon?: IconName;
  style?: ViewStyle;
  compact?: boolean;
};

export function Button({ label, variant = "primary", loading, icon, disabled, style, compact, ...rest }: ButtonProps) {
  const { colors } = useTheme();
  const palette = {
    primary: { bg: colors.primary, fg: "onPrimary" as const, border: colors.primary },
    secondary: { bg: colors.surfaceAlt, fg: "text" as const, border: colors.border },
    ghost: { bg: "transparent", fg: "primary" as const, border: "transparent" },
    danger: { bg: colors.dangerSoft, fg: "danger" as const, border: colors.dangerSoft }
  }[variant];
  const inactive = disabled || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: Boolean(inactive), busy: Boolean(loading) }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        compact && styles.compact,
        { backgroundColor: palette.bg, borderColor: palette.border, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        pressed && !inactive ? styles.pressed : null,
        style
      ]}
      {...rest}
    >
      {loading ? (
        <ActivityIndicator color={colors[palette.fg]} />
      ) : (
        <View style={styles.row}>
          {icon ? <Icon name={icon} size={18} color={palette.fg} /> : null}
          <Text variant="label" color={palette.fg} style={styles.label}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.lg
  },
  compact: { minHeight: 40, paddingHorizontal: spacing.md },
  pressed: { transform: [{ scale: 0.98 }] },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  label: { fontSize: 15 }
});
