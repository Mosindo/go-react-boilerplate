import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { typography, useTheme, type Palette, type TypographyVariant } from "../theme";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "inverse";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  center?: boolean;
};

function toneColor(tone: TextTone, colors: Palette): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "subtle":
      return colors.textSubtle;
    case "primary":
      return colors.primary;
    case "accent":
      return colors.accent;
    case "danger":
      return colors.danger;
    case "success":
      return colors.success;
    case "inverse":
      return "#FFFFFF";
    default:
      return colors.text;
  }
}

export function Text({ variant = "body", tone = "default", center, style, ...props }: TextProps) {
  const { colors } = useTheme();
  return (
    <RNText {...props} style={[typography[variant], { color: toneColor(tone, colors) }, center ? { textAlign: "center" } : null, style]} />
  );
}
