import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { fontSizes, fontWeights, lineHeights, useColors, type Palette } from "../theme";

export type TextVariant = "display" | "heading" | "title" | "body" | "small" | "caption";
export type TextTone = "default" | "muted" | "primary" | "danger" | "success" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: keyof typeof fontWeights;
  center?: boolean;
};

function toneColor(tone: TextTone, colors: Palette): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "primary":
      return colors.primary;
    case "danger":
      return colors.danger;
    case "success":
      return colors.success;
    case "onPrimary":
      return colors.onPrimary;
    default:
      return colors.text;
  }
}

const defaultWeight: Record<TextVariant, keyof typeof fontWeights> = {
  display: "bold",
  heading: "bold",
  title: "semibold",
  body: "regular",
  small: "regular",
  caption: "regular"
};

export function Text({ variant = "body", tone = "default", weight, center, style, ...rest }: TextProps) {
  const colors = useColors();
  return (
    <RNText
      accessibilityRole={variant === "display" || variant === "heading" ? "header" : rest.accessibilityRole}
      {...rest}
      style={[
        {
          color: toneColor(tone, colors),
          fontSize: fontSizes[variant],
          lineHeight: lineHeights[variant],
          fontWeight: fontWeights[weight ?? defaultWeight[variant]],
          textAlign: center ? "center" : undefined
        },
        style
      ]}
    />
  );
}
