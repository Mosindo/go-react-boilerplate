import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme, type TypographyVariant } from "../theme";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  center?: boolean;
};

export function Text({ variant = "body", tone = "default", center, style, ...props }: TextProps) {
  const { colors, typography } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    accent: colors.accent,
    danger: colors.danger,
    success: colors.success,
    onPrimary: colors.onPrimary
  }[tone];
  return <RNText {...props} style={[typography[variant], { color }, center ? { textAlign: "center" } : null, style]} />;
}
