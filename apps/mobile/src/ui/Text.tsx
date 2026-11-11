import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import type { TypeVariant } from "../theme/palette";

type Tone = "default" | "muted" | "subtle" | "primary" | "danger" | "success" | "inverse" | "onPrimary";

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  tone?: Tone;
  align?: "left" | "center" | "right";
  weight?: "400" | "500" | "600" | "700";
};

export function Text({ variant = "body", tone = "default", align, weight, style, ...rest }: TextProps) {
  const { colors, type } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    danger: colors.danger,
    success: colors.success,
    inverse: "#FFFFFF",
    onPrimary: colors.onPrimary
  }[tone];
  const t = type[variant];
  return (
    <RNText
      {...rest}
      style={[
        t,
        { color, textAlign: align },
        variant === "overline" ? { textTransform: "uppercase" } : null,
        weight ? { fontWeight: weight } : null,
        style
      ]}
    />
  );
}
