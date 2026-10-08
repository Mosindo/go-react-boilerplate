import React from "react";
import { Text as NativeText, type TextProps as NativeTextProps } from "react-native";
import { useTheme } from "../../theme";
import { fontWeights, typography } from "./tokens";

export type TextVariant = keyof typeof typography;
export type TextTone = "default" | "muted" | "subtle" | "primary" | "danger" | "success" | "inverse" | "onPrimary";

export type TextProps = NativeTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: keyof typeof fontWeights;
  align?: "left" | "center" | "right";
};

const defaultWeight: Partial<Record<TextVariant, keyof typeof fontWeights>> = {
  title: "bold",
  heading: "bold",
  label: "semibold",
  eyebrow: "semibold",
  button: "semibold"
};

export function Text({ variant = "body", tone = "default", weight, align, style, ...props }: TextProps) {
  const { colors } = useTheme();
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
  return (
    <NativeText
      style={[
        typography[variant],
        { color, fontWeight: fontWeights[weight ?? defaultWeight[variant] ?? "regular"] },
        align ? { textAlign: align } : null,
        style
      ]}
      {...props}
    />
  );
}
