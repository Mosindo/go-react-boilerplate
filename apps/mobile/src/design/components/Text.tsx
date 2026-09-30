import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "../ThemeProvider";
import type { TypographyVariant } from "../theme";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "danger" | "inverse" | "success";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  align?: "left" | "center" | "right";
};

export function Text({ variant = "body", tone = "default", align, style, ...props }: TextProps) {
  const theme = useTheme();
  const color = {
    default: theme.colors.text,
    muted: theme.colors.textMuted,
    subtle: theme.colors.textSubtle,
    primary: theme.colors.primary,
    danger: theme.colors.danger,
    inverse: theme.colors.onPrimary,
    success: theme.colors.success
  }[tone];
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      style={[theme.typography[variant], { color, textAlign: align }, style]}
      {...props}
    />
  );
}
