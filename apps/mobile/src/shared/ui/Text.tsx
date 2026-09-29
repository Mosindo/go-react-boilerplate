import React from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "./theme";

export type TextVariant = "body" | "label" | "title" | "heading" | "caption" | "eyebrow" | "button";
export type TextTone = "default" | "muted" | "subtle" | "primary" | "secondary" | "danger" | "success" | "inverse";
export type TextWeight = "regular" | "medium" | "semibold" | "bold";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

/** Caps runaway Dynamic Type so layouts survive, while still honouring the user's setting. */
const MAX_FONT_SCALE = 1.6;

export function Text({
  children,
  maxFontSizeMultiplier = MAX_FONT_SCALE,
  style,
  tone = "default",
  variant = "body",
  weight = "regular",
  ...props
}: TextProps) {
  const theme = useTheme();
  const toneColors: Record<TextTone, string> = {
    default: theme.colors.text,
    muted: theme.colors.textMuted,
    subtle: theme.colors.textSubtle,
    primary: theme.colors.primary,
    secondary: theme.colors.secondary,
    danger: theme.colors.danger,
    success: theme.colors.success,
    inverse: theme.colors.primaryForeground
  };
  return (
    <RNText
      maxFontSizeMultiplier={maxFontSizeMultiplier}
      style={[
        theme.typography[variant] as TextStyle,
        { fontWeight: theme.fontWeights[weight], color: toneColors[tone] },
        style
      ]}
      {...props}
    >
      {children}
    </RNText>
  );
}
