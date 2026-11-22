import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { colors, typography } from "./theme";

export type TextVariant = keyof typeof typography;
export type TextTone = "default" | "muted" | "subtle" | "accent" | "like" | "danger" | "onAccent";

const toneColor: Record<TextTone, string> = {
  default: colors.text,
  muted: colors.textMuted,
  subtle: colors.textSubtle,
  accent: colors.accent,
  like: colors.like,
  danger: colors.danger,
  onAccent: colors.textOnAccent
};

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  align?: TextStyle["textAlign"];
};

export function Text({ variant = "body", tone = "default", align, style, ...props }: TextProps) {
  const base = typography[variant];
  return (
    <RNText
      {...props}
      style={[
        base as TextStyle,
        { color: toneColor[tone], textAlign: align },
        variant === "eyebrow" ? { textTransform: "uppercase" } : null,
        style
      ]}
    />
  );
}
