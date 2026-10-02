import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";

import { useTheme } from "../theme";

export type TextVariant = "display" | "title" | "heading" | "body" | "label" | "caption";
type Tone = "default" | "muted" | "primary" | "danger" | "onPrimary";

const sizes: Record<
  TextVariant,
  { fontSize: number; lineHeight: number; fontWeight: "400" | "600" | "700" | "800" }
> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800" },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
};

export interface TextProps extends RNTextProps {
  variant?: TextVariant;
  tone?: Tone;
  center?: boolean;
}

export function Text({ variant = "body", tone = "default", center, style, ...props }: TextProps) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    primary: colors.primary,
    danger: colors.danger,
    onPrimary: colors.onPrimary,
  }[tone];
  return (
    <RNText
      accessibilityRole={variant === "display" || variant === "title" ? "header" : undefined}
      style={[sizes[variant], { color }, center && { textAlign: "center" }, style]}
      {...props}
    />
  );
}
