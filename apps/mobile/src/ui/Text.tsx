import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { fontSizes, useTheme, type Palette } from "../theme/theme";

export type TextVariant = "display" | "title" | "heading" | "body" | "label" | "caption";

const variants: Record<TextVariant, TextStyle> = {
  display: { fontSize: fontSizes.xxl, lineHeight: 38, fontWeight: "700", letterSpacing: -0.6 },
  title: { fontSize: fontSizes.xl, lineHeight: 30, fontWeight: "700", letterSpacing: -0.3 },
  heading: { fontSize: fontSizes.lg, lineHeight: 24, fontWeight: "600" },
  body: { fontSize: fontSizes.md, lineHeight: 22, fontWeight: "400" },
  label: { fontSize: fontSizes.sm, lineHeight: 18, fontWeight: "600" },
  caption: { fontSize: fontSizes.xs, lineHeight: 16, fontWeight: "400" }
};

type Props = TextProps & {
  variant?: TextVariant;
  tone?: keyof Pick<
    Palette,
    "text" | "textMuted" | "textSubtle" | "primary" | "danger" | "onPrimary" | "success"
  >;
  align?: TextStyle["textAlign"];
};

export function Text({ variant = "body", tone = "text", align, style, ...rest }: Props) {
  const { colors } = useTheme();
  return <RNText {...rest} style={[variants[variant], { color: colors[tone], textAlign: align }, style]} />;
}
