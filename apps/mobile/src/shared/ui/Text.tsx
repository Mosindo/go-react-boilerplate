import React from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../../theme";
import { fontWeights, typography, type Colors, type TypographyToken } from "./tokens";

export type TextVariant = TypographyToken;
export type TextTone = "default" | "muted" | "subtle" | "primary" | "accent" | "danger" | "success" | "inverse";
export type TextWeight = keyof typeof fontWeights;

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

function toneColor(colors: Colors, tone: TextTone): string {
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

export function Text({ children, style, tone = "default", variant = "body", weight = "regular", ...props }: TextProps) {
  const { colors } = useTheme();
  return (
    <RNText style={[typography[variant], { fontWeight: fontWeights[weight], color: toneColor(colors, tone) }, style]} {...props}>
      {children}
    </RNText>
  );
}
