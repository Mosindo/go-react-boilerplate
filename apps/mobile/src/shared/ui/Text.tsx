import React from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "./theme";
import { fontWeights, typography, type Palette } from "./tokens";

export type TextVariant = "body" | "label" | "title" | "heading" | "caption" | "eyebrow" | "button";
export type TextTone = "default" | "muted" | "subtle" | "primary" | "secondary" | "danger" | "success" | "inverse";
export type TextWeight = "regular" | "medium" | "semibold" | "bold";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

function toneColor(colors: Palette, tone: TextTone): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "subtle":
      return colors.textSubtle;
    case "primary":
      return colors.primary;
    case "secondary":
      return colors.secondary;
    case "danger":
      return colors.danger;
    case "success":
      return colors.success;
    case "inverse":
      return colors.inverse;
    default:
      return colors.text;
  }
}

export function Text({ variant = "body", tone = "default", weight, style, ...props }: TextProps) {
  const { colors } = useTheme();
  const defaultWeight: TextWeight = variant === "title" || variant === "heading" ? "bold" : variant === "button" ? "semibold" : "regular";
  return (
    <RNText
      {...props}
      style={[typography[variant], { color: toneColor(colors, tone), fontWeight: fontWeights[weight ?? defaultWeight] }, style]}
    />
  );
}
