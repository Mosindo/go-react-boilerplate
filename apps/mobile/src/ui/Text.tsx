import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme";

type Variant = "display" | "title" | "heading" | "body" | "caption" | "label";
type Tone = "default" | "muted" | "primary" | "danger" | "onPrimary";

const variants: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700", letterSpacing: -0.5 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.3 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "600" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 18, fontWeight: "600" }
};

export type TextProps = RNTextProps & { variant?: Variant; tone?: Tone };

export function Text({ variant = "body", tone = "default", style, ...rest }: TextProps) {
  const t = useTheme();
  const color = {
    default: t.text,
    muted: t.textMuted,
    primary: t.primary,
    danger: t.danger,
    onPrimary: t.onPrimary
  }[tone];
  return <RNText {...rest} style={[variants[variant], { color }, style]} />;
}
