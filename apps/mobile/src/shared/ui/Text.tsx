import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { typography, useTheme } from "../../theme";

export type TextVariant = keyof typeof typography;
export type TextTone = "default" | "muted" | "primary" | "danger" | "success" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  center?: boolean;
};

export function Text({ variant = "body", tone = "default", center = false, style, ...rest }: TextProps) {
  const theme = useTheme();
  const colorByTone: Record<TextTone, string> = {
    default: theme.text,
    muted: theme.textMuted,
    primary: theme.primary,
    danger: theme.danger,
    success: theme.success,
    onPrimary: theme.onPrimary
  };
  return (
    <RNText
      accessibilityRole={variant === "title" || variant === "display" || variant === "heading" ? "header" : undefined}
      style={[typography[variant], { color: colorByTone[tone] }, center ? { textAlign: "center" } : null, style]}
      {...rest}
    />
  );
}
