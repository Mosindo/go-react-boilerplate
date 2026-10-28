import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { fontSizes, useTheme } from "../theme";

type Variant = "title" | "heading" | "body" | "caption" | "label";
type Tone = "default" | "muted" | "primary" | "danger" | "inverse";

const variantStyle: Record<Variant, TextStyle> = {
  title: { fontSize: fontSizes.xl, fontWeight: "700", letterSpacing: -0.4 },
  heading: { fontSize: fontSizes.lg, fontWeight: "700" },
  body: { fontSize: fontSizes.md, lineHeight: 22 },
  caption: { fontSize: fontSizes.sm, lineHeight: 18 },
  label: { fontSize: fontSizes.sm, fontWeight: "600" }
};

type Props = TextProps & { variant?: Variant; tone?: Tone };

export function Text({ variant = "body", tone = "default", style, ...rest }: Props) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    primary: colors.primary,
    danger: colors.danger,
    inverse: "#FFFFFF"
  }[tone];
  return <RNText accessibilityRole={variant === "title" || variant === "heading" ? "header" : undefined} style={[{ color }, variantStyle[variant], style]} {...rest} />;
}
