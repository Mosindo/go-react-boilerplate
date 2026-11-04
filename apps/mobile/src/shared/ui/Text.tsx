import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { useColors, type Colors } from "./theme";

export type TextVariant = "title" | "heading" | "body" | "label" | "caption";
type Tone = "default" | "muted" | "primary" | "danger" | "accent" | "onPrimary";

const variants: Record<TextVariant, TextStyle> = {
  title: { fontSize: 30, lineHeight: 36, fontWeight: "700", letterSpacing: -0.4 },
  heading: { fontSize: 20, lineHeight: 26, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 22 },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: 12, lineHeight: 16 },
};

function toneColor(tone: Tone, c: Colors): string {
  switch (tone) {
    case "muted":
      return c.textMuted;
    case "primary":
      return c.primary;
    case "danger":
      return c.danger;
    case "accent":
      return c.accent;
    case "onPrimary":
      return c.onPrimary;
    default:
      return c.text;
  }
}

type Props = TextProps & { variant?: TextVariant; tone?: Tone };

export function Text({ variant = "body", tone = "default", style, ...rest }: Props) {
  const colors = useColors();
  return <RNText accessibilityRole={variant === "title" || variant === "heading" ? "header" : undefined} style={[variants[variant], { color: toneColor(tone, colors) }, style]} {...rest} />;
}
