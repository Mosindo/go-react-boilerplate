import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { typography, useTheme, type Palette, type TypographyVariant } from "../theme";

export type TextProps = RNTextProps & {
  variant?: TypographyVariant;
  color?: keyof Palette;
  align?: "left" | "center" | "right";
};

export function Text({ variant = "body", color = "text", align, style, ...rest }: TextProps) {
  const { colors } = useTheme();
  return <RNText {...rest} style={[typography[variant], { color: colors[color] }, align ? { textAlign: align } : null, style]} />;
}
