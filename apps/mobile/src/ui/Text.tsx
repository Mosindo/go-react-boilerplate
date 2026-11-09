import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme/theme";

export type TextVariant = "title" | "heading" | "body" | "caption" | "label";

const variants: Record<TextVariant, TextStyle> = {
  title: { fontSize: 30, fontWeight: "800", letterSpacing: -0.5 },
  heading: { fontSize: 20, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23 },
  caption: { fontSize: 13, lineHeight: 18 },
  label: { fontSize: 14, fontWeight: "600" }
};

type Props = TextProps & { variant?: TextVariant; muted?: boolean; color?: string };

export function Text({ variant = "body", muted, color, style, ...rest }: Props) {
  const { colors } = useTheme();
  return <RNText {...rest} style={[variants[variant], { color: color ?? (muted ? colors.textMuted : colors.text) }, style]} />;
}
