import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme } from "../theme";

type Variant = "title" | "heading" | "body" | "caption" | "label";
type Tone = "default" | "muted" | "primary" | "danger" | "onPrimary";

const sizes: Record<
  Variant,
  { fontSize: number; fontWeight: "400" | "600" | "700" | "800"; lineHeight: number }
> = {
  title: { fontSize: 30, fontWeight: "800", lineHeight: 36 },
  heading: { fontSize: 20, fontWeight: "700", lineHeight: 26 },
  body: { fontSize: 16, fontWeight: "400", lineHeight: 23 },
  label: { fontSize: 15, fontWeight: "600", lineHeight: 20 },
  caption: { fontSize: 13, fontWeight: "400", lineHeight: 18 }
};

export type TextProps = RNTextProps & { variant?: Variant; tone?: Tone };

export function Text({ variant = "body", tone = "default", style, ...rest }: TextProps) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    primary: colors.primary,
    danger: colors.danger,
    onPrimary: colors.onPrimary
  }[tone];
  return <RNText {...rest} style={[sizes[variant], { color }, style]} />;
}
