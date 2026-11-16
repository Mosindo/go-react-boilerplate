import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { colors, typography } from "../theme/tokens";

type Variant = keyof typeof typography;
type Tone = "default" | "muted" | "subtle" | "accent" | "danger" | "success" | "onAccent";

const toneColor: Record<Tone, string> = {
  default: colors.text,
  muted: colors.textMuted,
  subtle: colors.textSubtle,
  accent: colors.accent,
  danger: colors.danger,
  success: colors.success,
  onAccent: colors.textOnAccent
};

export type TextProps = RNTextProps & {
  variant?: Variant;
  tone?: Tone;
  align?: TextStyle["textAlign"];
};

export function Text({ variant = "body", tone = "default", align, style, ...rest }: TextProps) {
  return (
    <RNText
      accessibilityRole={variant === "display" || variant === "title" || variant === "heading" ? "header" : undefined}
      {...rest}
      style={[typography[variant], { color: toneColor[tone], textAlign: align }, style]}
    />
  );
}
