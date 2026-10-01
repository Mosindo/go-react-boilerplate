import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "./theme";
import { fontWeights, typography, type ColorToken, type TypographyToken } from "./tokens";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "secondary" | "danger" | "success" | "inverse";
export type TextWeight = keyof typeof fontWeights;

export type TextProps = RNTextProps & {
  variant?: TypographyToken;
  tone?: TextTone;
  weight?: TextWeight;
  align?: "left" | "center" | "right";
};

const toneToColor: Record<TextTone, ColorToken> = {
  default: "text",
  muted: "textMuted",
  subtle: "textSubtle",
  primary: "primary",
  secondary: "secondary",
  danger: "danger",
  success: "success",
  inverse: "inverse"
};

export function Text({ variant = "body", tone = "default", weight, align, style, ...props }: TextProps) {
  const { colors } = useTheme();
  const defaultWeight: TextWeight = variant === "title" || variant === "heading" || variant === "button" ? "bold" : "regular";
  return (
    <RNText
      {...props}
      style={[
        typography[variant],
        { color: colors[toneToColor[tone]], fontWeight: fontWeights[weight ?? defaultWeight], textAlign: align },
        style
      ]}
    />
  );
}
