import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { colors, typography, type TypographyVariant } from "../theme";

type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  color?: keyof typeof colors;
  align?: TextStyle["textAlign"];
};

export function Text({ variant = "body", color = "text", align, style, ...rest }: TextProps) {
  return <RNText {...rest} style={[typography[variant], { color: colors[color], textAlign: align }, style]} />;
}
