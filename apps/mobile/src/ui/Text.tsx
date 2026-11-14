import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { type, type TypeVariant } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "inverse" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  tone?: TextTone;
  center?: boolean;
  weight?: "400" | "500" | "600" | "700";
};

export function Text({ variant = "body", tone = "default", center, weight, style, ...rest }: TextProps) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    accent: colors.accent,
    danger: colors.danger,
    success: colors.success,
    inverse: colors.textInverse,
    onPrimary: colors.onPrimary
  }[tone];
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      style={[type[variant], { color }, center ? { textAlign: "center" } : null, weight ? { fontWeight: weight } : null, style]}
      {...rest}
    />
  );
}
