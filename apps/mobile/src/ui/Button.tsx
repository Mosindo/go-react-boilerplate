import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { layout, radii, spacing, typography, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  icon?: IconName;
  loading?: boolean;
  fullWidth?: boolean;
  size?: "md" | "sm";
  style?: StyleProp<ViewStyle>;
};

export function Button({
  label,
  variant = "primary",
  icon,
  loading,
  disabled,
  fullWidth = true,
  size = "md",
  style,
  ...rest
}: ButtonProps) {
  const { colors } = useTheme();
  const palette = {
    primary: { bg: colors.accent, pressed: colors.accentPressed, fg: "textOnAccent" as const, border: colors.accent },
    secondary: { bg: colors.surface, pressed: colors.surfaceAlt, fg: "text" as const, border: colors.borderStrong },
    ghost: { bg: "transparent", pressed: colors.surfaceAlt, fg: "accent" as const, border: "transparent" },
    danger: { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: "danger" as const, border: colors.dangerSoft }
  }[variant];
  const isDisabled = disabled || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      accessibilityLabel={label}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        size === "sm" ? styles.small : null,
        { backgroundColor: pressed ? palette.pressed : palette.bg, borderColor: palette.border },
        fullWidth ? styles.full : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...rest}
    >
      {loading ? (
        <ActivityIndicator color={colors[palette.fg]} />
      ) : (
        <View style={styles.content}>
          {icon ? <Icon color={palette.fg} name={icon} size={size === "sm" ? 16 : 19} /> : null}
          <Text color={palette.fg} style={size === "sm" ? styles.smallLabel : styles.label}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: layout.touchTarget + 4,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center"
  },
  small: { minHeight: 38, paddingHorizontal: spacing.md },
  full: { alignSelf: "stretch" },
  disabled: { opacity: 0.5 },
  content: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  label: { ...typography.bodyStrong },
  smallLabel: { ...typography.label, fontWeight: "600" }
});
