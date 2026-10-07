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
import { useTheme } from "../../theme/useTheme";
import type { ColorPalette } from "../../theme/palette";
import { Text, type TextTone } from "./Text";
import { controls, radii, shadows, spacing } from "./tokens";

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
  base: { borderRadius: radii.pill, alignItems: "center", justifyContent: "center" },
  fullWidth: { width: "100%" },
  small: { minHeight: controls.button.sm, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  medium: { minHeight: controls.button.md, paddingHorizontal: spacing.xl, paddingVertical: spacing.md },
  large: { minHeight: controls.button.lg, paddingHorizontal: spacing.xl, paddingVertical: spacing.lg },
  disabled: { opacity: 0.5 },
  pressed: { opacity: 0.88 },
  content: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.sm }
});

const sizeStyles: Record<ButtonSize, ViewStyle> = { sm: styles.small, md: styles.medium, lg: styles.large };

function variantStyle(colors: ColorPalette, variant: ButtonVariant): ViewStyle {
  switch (variant) {
    case "primary":
      return { backgroundColor: colors.primary, ...shadows.card };
    case "secondary":
      return { backgroundColor: colors.primarySoft, borderWidth: 1, borderColor: colors.primaryBorder };
    case "destructive":
      return { backgroundColor: colors.danger, borderWidth: 1, borderColor: colors.danger };
    case "success":
      return { backgroundColor: colors.success, borderWidth: 1, borderColor: colors.success };
    case "outline":
      return { backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.borderStrong };
    default:
      return { backgroundColor: "transparent" };
  }
}

const textToneByVariant: Record<ButtonVariant, TextTone> = {
  primary: "inverse",
  secondary: "primary",
  destructive: "inverse",
  success: "inverse",
  outline: "default",
  ghost: "secondary"
};

function spinnerColor(colors: ColorPalette, variant: ButtonVariant): string {
  switch (variant) {
    case "primary":
    case "destructive":
    case "success":
      return colors.inverse;
    case "secondary":
      return colors.primary;
    default:
      return colors.secondary;
  }
}

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
  const { colors } = useTheme();
  const buttonDisabled = disabled || loading;
  const labelNode = typeof children === "string" ? children : label;
  // Primary uses `primaryForeground` for its label, which differs from `inverse` only in naming.
  const primaryLabel = variant === "primary" ? { color: colors.primaryForeground } : null;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: !!buttonDisabled, busy: loading }}
      disabled={buttonDisabled}
      style={({ pressed }) => [
        styles.base,
        sizeStyles[size],
        variantStyle(colors, variant),
        fullWidth ? styles.fullWidth : null,
        buttonDisabled ? styles.disabled : null,
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={spinnerColor(colors, variant)} size="small" /> : null}
        {labelNode ? (
          <Text
            style={[primaryLabel, textStyle]}
            tone={textToneByVariant[variant]}
            variant="button"
            weight="bold"
          >
            {labelNode}
          </Text>
        ) : (
          children
        )}
      </View>
    </Pressable>
  );
}
