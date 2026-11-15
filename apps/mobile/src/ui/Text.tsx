import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { type, useTheme, type TypeVariant, type Palette } from "../theme";

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  color?: keyof Palette;
  align?: "left" | "center" | "right";
};

export function Text({ variant = "body", color = "text", align, style, ...rest }: TextProps) {
  const { colors } = useTheme();
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      {...rest}
      style={[type[variant], { color: colors[color] }, align ? { textAlign: align } : null, style]}
    />
  );
}
