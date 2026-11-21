import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme, type ThemeColors, type TypographyVariant } from "../theme";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "danger" | "success" | "onPrimary" | "onImage";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  align?: "left" | "center" | "right";
};

const toneColor: Record<TextTone, keyof ThemeColors> = {
  default: "text",
  muted: "textMuted",
  subtle: "textSubtle",
  primary: "primary",
  danger: "danger",
  success: "success",
  onPrimary: "textOnPrimary",
  onImage: "textOnImage"
};

export function Text({ variant = "body", tone = "default", align, style, ...props }: TextProps) {
  const theme = useTheme();
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      style={[theme.typography[variant], { color: theme.colors[toneColor[tone]], textAlign: align }, style]}
      {...props}
    />
  );
}
