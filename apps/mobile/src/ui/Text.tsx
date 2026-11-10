import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import type { Palette, TypographyVariant } from "../theme/tokens";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "danger" | "success" | "inverse" | "onPrimary" | "accent";

const toneToColor: Record<TextTone, keyof Palette> = {
  default: "text",
  muted: "textMuted",
  subtle: "textSubtle",
  primary: "primary",
  danger: "danger",
  success: "success",
  inverse: "textInverse",
  onPrimary: "onPrimary",
  accent: "accent"
};

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  tone?: TextTone;
  weight?: "400" | "500" | "600" | "700";
  align?: "left" | "center" | "right";
  color?: string;
};

export function Text({ variant = "body", tone = "default", weight, align, color, style, ...props }: TextProps) {
  const theme = useTheme();
  return (
    <RNText
      {...props}
      style={[
        theme.typography[variant],
        { color: color ?? theme.colors[toneToColor[tone]] },
        weight ? { fontWeight: weight } : null,
        align ? { textAlign: align } : null,
        style
      ]}
    />
  );
}
