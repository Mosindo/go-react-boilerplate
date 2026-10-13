import React, { type ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { controls, radii, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  icon?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Button({
  children,
  disabled,
  fullWidth = false,
  icon,
  label,
  loading = false,
  size = "md",
  style,
  variant = "primary",
  accessibilityLabel,
  ...props
}: ButtonProps) {
  const { colors } = useTheme();
  const inactive = disabled || loading;
  const text = typeof children === "string" ? children : label;

  const palette = {
    primary: { bg: colors.primary, fg: colors.primaryForeground, border: colors.primary },
    secondary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primarySoft },
    outline: { bg: "transparent", fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.textMuted, border: "transparent" },
    destructive: { bg: colors.danger, fg: "#FFFFFF", border: colors.danger }
  }[variant];

  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? text}
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: controls.button[size],
          backgroundColor: palette.bg,
          borderColor: palette.border,
          opacity: inactive ? 0.55 : pressed ? 0.85 : 1,
          paddingHorizontal: size === "sm" ? spacing.lg : spacing.xl
        },
        fullWidth ? styles.fullWidth : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={palette.fg} size="small" /> : icon}
        {text ? (
          <Text style={{ color: palette.fg }} variant="button" weight="semibold">
            {text}
          </Text>
        ) : (
          children
        )}
      </View>
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
  fullWidth: { width: "100%" },
  content: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.sm }
});
