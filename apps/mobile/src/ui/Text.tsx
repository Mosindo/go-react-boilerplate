import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";

export type TextVariant = "display" | "title" | "heading" | "body" | "label" | "caption";
export type TextTone =
  "default" | "muted" | "primary" | "accent" | "danger" | "onPrimary" | "onAccent";

const sizes: Record<TextVariant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800" },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "800" },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 15, lineHeight: 20, fontWeight: "700" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" }
};

type Props = TextProps & { variant?: TextVariant; tone?: TextTone; center?: boolean };

export function Text({ variant = "body", tone = "default", center, style, ...rest }: Props) {
  const { colors, fontFamily } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    primary: colors.primary,
    accent: colors.accent,
    danger: colors.danger,
    onPrimary: colors.onPrimary,
    onAccent: colors.onAccent
  }[tone];
  return (
    <RNText
      accessibilityRole={variant === "display" || variant === "title" ? "header" : undefined}
      {...rest}
      style={[
        { fontFamily, color },
        sizes[variant],
        center ? { textAlign: "center" } : null,
        style
      ]}
    />
  );
}
