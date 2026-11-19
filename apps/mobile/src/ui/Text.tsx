import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { fonts, type, type Palette, type TypeVariant } from "../theme/tokens";

type Tone = "default" | "muted" | "subtle" | "primary" | "danger" | "onPrimary" | "onPhoto" | "success";

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  tone?: Tone;
  align?: "left" | "center" | "right";
  weight?: "400" | "500" | "600" | "700";
};

const toneColor = (colors: Palette, tone: Tone): string =>
  ({
    default: colors.text,
    muted: colors.textMuted,
    subtle: colors.textSubtle,
    primary: colors.primary,
    danger: colors.danger,
    onPrimary: colors.onPrimary,
    onPhoto: colors.onPhoto,
    success: colors.success
  })[tone];

export function Text({ variant = "body", tone = "default", align, weight, style, ...props }: TextProps) {
  const { colors } = useTheme();
  const serif = variant === "display" || variant === "title";
  return (
    <RNText
      {...props}
      style={[
        type[variant],
        { color: toneColor(colors, tone), fontFamily: serif ? fonts.display : fonts.body },
        align ? { textAlign: align } : null,
        weight ? { fontWeight: weight } : null,
        style
      ]}
    />
  );
}
