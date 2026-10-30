import React from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme, type Palette } from "./theme";

type Variant = "display" | "title" | "heading" | "body" | "label" | "caption";
type Tone = "default" | "muted" | "primary" | "danger" | "inverse";

const variants: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800", letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.4 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" }
};

function toneColor(tone: Tone, colors: Palette): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "primary":
      return colors.primary;
    case "danger":
      return colors.danger;
    case "inverse":
      return "#FFFFFF";
    default:
      return colors.text;
  }
}

export type TextProps = RNTextProps & {
  variant?: Variant;
  tone?: Tone;
  style?: StyleProp<TextStyle>;
};

export function Text({ variant = "body", tone = "default", style, ...props }: TextProps) {
  const { colors } = useTheme();
  return <RNText style={[variants[variant], { color: toneColor(tone, colors) }, style]} {...props} />;
}
