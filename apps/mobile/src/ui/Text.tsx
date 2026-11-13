import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";

type Variant = "display" | "title" | "heading" | "body" | "label" | "caption";
type Tone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: Variant;
  tone?: Tone;
  weight?: "regular" | "medium" | "semibold" | "bold";
  align?: TextStyle["textAlign"];
};

const variants: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, letterSpacing: -0.6 },
  title: { fontSize: 26, lineHeight: 32, letterSpacing: -0.4 },
  heading: { fontSize: 19, lineHeight: 25, letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23 },
  label: { fontSize: 14, lineHeight: 19 },
  caption: { fontSize: 12.5, lineHeight: 17 }
};

const weights = { regular: "400", medium: "500", semibold: "600", bold: "700" } as const;

export function Text({ variant = "body", tone = "default", weight, align, style, ...props }: TextProps) {
  const { colors, displayFont } = useTheme();
  const color: Record<Tone, string> = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    accent: colors.accent,
    danger: colors.danger,
    onPrimary: colors.onPrimary
  };
  const isDisplay = variant === "display" || variant === "title";
  return (
    <RNText
      {...props}
      style={[
        variants[variant],
        {
          color: color[tone],
          fontWeight: weights[weight ?? (isDisplay ? "regular" : variant === "heading" ? "semibold" : "regular")],
          textAlign: align
        },
        isDisplay ? { fontFamily: displayFont } : null,
        style
      ]}
    />
  );
}
