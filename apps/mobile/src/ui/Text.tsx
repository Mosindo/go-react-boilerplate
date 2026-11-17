import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { colors, typography, type TypographyVariant } from "../theme";

type Tone = "default" | "muted" | "subtle" | "accent" | "danger" | "success" | "inverse" | "gold";

const toneColor: Record<Tone, string> = {
  default: colors.text,
  muted: colors.textMuted,
  subtle: colors.textSubtle,
  accent: colors.accent,
  danger: colors.danger,
  success: colors.success,
  inverse: colors.accentText,
  gold: colors.gold
};

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: Tone;
  center?: boolean;
};

export function Text({ variant = "body", tone = "default", center, style, ...props }: TextProps) {
  return <RNText {...props} style={[typography[variant], { color: toneColor[tone] }, center ? { textAlign: "center" } : null, style]} />;
}
