import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { fontSize, useTheme, type ColorTokens } from "../../theme";

export type TextVariant = "title" | "heading" | "lead" | "body" | "label" | "caption";

const VARIANTS: Record<TextVariant, TextStyle> = {
  title: { fontSize: fontSize.title, lineHeight: 38, fontWeight: "800", letterSpacing: -0.5 },
  heading: { fontSize: fontSize.heading, lineHeight: 28, fontWeight: "700", letterSpacing: -0.2 },
  lead: { fontSize: fontSize.lead, lineHeight: 26, fontWeight: "400" },
  body: { fontSize: fontSize.body, lineHeight: 22, fontWeight: "400" },
  label: { fontSize: fontSize.body, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: fontSize.caption, lineHeight: 18, fontWeight: "400" }
};

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  color?: keyof ColorTokens;
};

export function Text({ variant = "body", color = "text", style, ...rest }: TextProps) {
  const { colors } = useTheme();
  return <RNText {...rest} style={[VARIANTS[variant], { color: colors[color] }, style]} />;
}
