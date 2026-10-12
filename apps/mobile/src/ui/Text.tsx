import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme, type Palette } from "../theme";

type Variant = "display" | "title" | "heading" | "body" | "label" | "caption";
type Tone = "default" | "muted" | "primary" | "danger" | "onPrimary" | "success";

const sizes: Record<Variant, { fontSize: number; lineHeight: number; fontWeight: "400" | "500" | "600" | "700" | "800" }> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800" },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" },
  heading: { fontSize: 19, lineHeight: 26, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: 12, lineHeight: 16, fontWeight: "500" }
};

function toneColor(colors: Palette, tone: Tone): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "primary":
      return colors.primary;
    case "danger":
      return colors.danger;
    case "onPrimary":
      return colors.onPrimary;
    case "success":
      return colors.success;
    default:
      return colors.text;
  }
}

export type TextProps = RNTextProps & { variant?: Variant; tone?: Tone };

export function Text({ variant = "body", tone = "default", style, ...props }: TextProps) {
  const { colors } = useTheme();
  return <RNText allowFontScaling style={[sizes[variant], { color: toneColor(colors, tone) }, style]} {...props} />;
}
