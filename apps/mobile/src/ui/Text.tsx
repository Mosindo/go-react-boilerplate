import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { typography, type TypographyVariant } from "../theme/palette";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "onPrimary" | "inverse";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  align?: TextStyle["textAlign"];
};

export function Text({ variant = "body", tone = "default", align, style, ...props }: TextProps) {
  const { colors } = useTheme();
  const toneColor: Record<TextTone, string> = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    accent: colors.accent,
    danger: colors.danger,
    success: colors.success,
    onPrimary: colors.onPrimary,
    inverse: "#FFFFFF"
  };
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      style={[typography[variant], { color: toneColor[tone] }, align ? { textAlign: align } : null, style]}
      {...props}
    />
  );
}
