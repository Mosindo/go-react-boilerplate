import React, { type ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, controls, radii, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  label: string;
  icon?: ReactNode;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
};

const containerByVariant: Record<ButtonVariant, ViewStyle> = {
  primary: { backgroundColor: colors.primary },
  secondary: { backgroundColor: colors.surfaceRaised, borderWidth: 1, borderColor: colors.border },
  ghost: { backgroundColor: "transparent" },
  danger: { backgroundColor: colors.dangerSoft, borderWidth: 1, borderColor: colors.dangerBorder }
};

const toneByVariant = {
  primary: "onPrimary",
  secondary: "default",
  ghost: "primary",
  danger: "danger"
} as const;

export function Button({
  disabled,
  fullWidth = false,
  icon,
  label,
  loading = false,
  size = "md",
  style,
  variant = "primary",
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading;
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: loading }}
      disabled={isDisabled}
      hitSlop={controls.hitSlop}
      style={({ pressed }) => [
        styles.base,
        { minHeight: controls.button[size], paddingHorizontal: size === "sm" ? spacing.lg : spacing.xl },
        containerByVariant[variant],
        fullWidth ? styles.fullWidth : null,
        pressed && variant === "primary" ? { backgroundColor: colors.primaryPressed } : null,
        pressed && variant !== "primary" ? styles.pressed : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={variant === "primary" ? colors.onPrimary : colors.primary} size="small" /> : icon}
        <Text tone={toneByVariant[variant]} variant={size === "sm" ? "label" : "button"} weight="semibold">
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center"
  },
  fullWidth: {
    alignSelf: "stretch"
  },
  pressed: {
    opacity: 0.75
  },
  disabled: {
    opacity: 0.45
  },
  content: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  }
});
