import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { type, useTheme, type TypeVariant } from "../theme";

export type TextTone = "default" | "muted" | "subtle" | "primary" | "danger" | "success" | "onPrimary" | "onPhoto";

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  tone?: TextTone;
  serif?: boolean;
  center?: boolean;
};

export function Text({ variant = "body", tone = "default", serif = false, center = false, style, ...rest }: TextProps) {
  const { colors, serif: serifFamily } = useTheme();
  const toneColor: Record<TextTone, string> = {
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    danger: colors.danger,
    success: colors.success,
    onPrimary: colors.onPrimary,
    onPhoto: colors.onPhoto
  };
  return (
    <RNText
      maxFontSizeMultiplier={1.6}
      style={[
        type[variant],
        { color: toneColor[tone] },
        serif && serifFamily ? { fontFamily: serifFamily, fontWeight: "700" } : null,
        center ? { textAlign: "center" } : null,
        variant === "overline" ? { textTransform: "uppercase" } : null,
        style
      ]}
      {...rest}
    />
  );
}
