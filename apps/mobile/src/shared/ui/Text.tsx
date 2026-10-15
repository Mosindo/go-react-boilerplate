import React from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../../theme";
import { fontWeights, typography, type TypographyToken } from "./tokens";

export type TextVariant = TypographyToken;
export type TextTone = "default" | "muted" | "subtle" | "primary" | "brand" | "danger" | "success" | "inverse";
export type TextWeight = keyof typeof fontWeights;

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

export function Text({ children, style, tone = "default", variant = "body", weight = "regular", ...props }: TextProps) {
  const palette = useTheme();
  const colorByTone: Record<TextTone, string> = {
    default: palette.text,
    muted: palette.textMuted,
    subtle: palette.textSubtle,
    primary: palette.primary,
    brand: palette.brand,
    danger: palette.danger,
    success: palette.success,
    inverse: "#FFFFFF"
  };
  return (
    <RNText
      allowFontScaling
      maxFontSizeMultiplier={1.4}
      style={[typography[variant], { color: colorByTone[tone], fontWeight: fontWeights[weight] }, style]}
      {...props}
    >
      {children}
    </RNText>
  );
}
