import React from "react";
import { StyleSheet, Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { colors, fontWeights, typography } from "./tokens";

export type TextVariant = keyof typeof typography;
export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "onPrimary";
export type TextWeight = keyof typeof fontWeights;

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  align?: TextStyle["textAlign"];
};

const toneColors: Record<TextTone, string> = {
  default: colors.text,
  muted: colors.textMuted,
  subtle: colors.textSubtle,
  primary: colors.primary,
  accent: colors.accent,
  danger: colors.danger,
  success: colors.success,
  onPrimary: colors.onPrimary
};

const defaultWeight: Partial<Record<TextVariant, TextWeight>> = {
  display: "bold",
  title: "bold",
  heading: "semibold",
  button: "semibold",
  eyebrow: "semibold"
};

export function Text({ align, style, tone = "default", variant = "body", weight, ...props }: TextProps) {
  return (
    <RNText
      style={[
        styles.base,
        typography[variant],
        { color: toneColors[tone], fontWeight: fontWeights[weight ?? defaultWeight[variant] ?? "regular"] },
        align ? { textAlign: align } : null,
        style
      ]}
      {...props}
    />
  );
}

const styles = StyleSheet.create({
  base: {
    color: colors.text
  }
});
