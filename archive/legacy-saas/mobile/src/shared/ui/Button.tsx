import React, { type ReactNode } from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  View,
  type PressableProps,
  type StyleProp,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { Text } from "./Text";
import { colors, controls, radii, shadows, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive" | "success";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
};

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  fullWidth: {
    width: "100%"
  },
  small: {
    minHeight: controls.button.sm,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  },
  medium: {
    minHeight: controls.button.md,
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.md
  },
  large: {
    minHeight: controls.button.lg,
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.lg
  },
  primary: {
    backgroundColor: colors.text,
    ...shadows.card
  },
  secondary: {
    backgroundColor: colors.primarySoft,
    borderWidth: 1,
    borderColor: colors.primaryBorder
  },
  destructive: {
    backgroundColor: colors.danger,
    borderWidth: 1,
    borderColor: colors.danger
  },
  success: {
    backgroundColor: colors.success,
    borderWidth: 1,
    borderColor: colors.success
  },
  outline: {
    backgroundColor: colors.backgroundElevated,
    borderWidth: 1,
    borderColor: colors.border
  },
  ghost: {
    backgroundColor: "transparent"
  },
  disabled: {
    opacity: 0.55
  },
  pressed: {
    opacity: 0.9
  },
  content: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm
  }
});

const sizeStyles: Record<ButtonSize, ViewStyle> = {
  sm: styles.small,
  md: styles.medium,
  lg: styles.large
};

const variantStyles: Record<ButtonVariant, ViewStyle> = {
  primary: styles.primary,
  secondary: styles.secondary,
  destructive: styles.destructive,
  success: styles.success,
  outline: styles.outline,
  ghost: styles.ghost
};

const textToneByVariant: Record<ButtonVariant, "default" | "inverse" | "primary" | "secondary"> = {
  primary: "inverse",
  secondary: "primary",
  destructive: "inverse",
  success: "inverse",
  outline: "default",
  ghost: "secondary"
};

const spinnerColorByVariant: Record<ButtonVariant, string> = {
  primary: colors.inverse,
  secondary: colors.primary,
  destructive: colors.inverse,
  success: colors.inverse,
  outline: colors.secondary,
  ghost: colors.secondary
};

export function Button({
  children,
  disabled,
  fullWidth = false,
  label,
  loading = false,
  size = "md",
  style,
  textStyle,
  variant = "primary",
  ...props
}: ButtonProps) {
  const buttonDisabled = disabled || loading;
  const textTone = textToneByVariant[variant];
  const labelNode = typeof children === "string" ? children : label;

  return (
    <Pressable
      accessibilityRole="button"
      disabled={buttonDisabled}
      style={({ pressed }) => [
        styles.base,
        sizeStyles[size],
        variantStyles[variant],
        fullWidth ? styles.fullWidth : null,
        buttonDisabled ? styles.disabled : null,
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={spinnerColorByVariant[variant]} size="small" /> : null}
        {labelNode ? (
          <Text style={textStyle} tone={textTone} variant="button" weight="bold">
            {labelNode}
          </Text>
        ) : (
          children
        )}
      </View>
    </Pressable>
  );
}
