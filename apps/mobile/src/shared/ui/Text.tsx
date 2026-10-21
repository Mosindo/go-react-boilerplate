import React from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { fontSize, useTheme } from "../../theme";

type Variant = "title" | "heading" | "body" | "caption" | "label";
type Tone = "default" | "muted" | "primary" | "danger" | "success" | "inverse";

export type TextProps = RNTextProps & { variant?: Variant; tone?: Tone; center?: boolean };

const sizes: Record<Variant, { fontSize: number; fontWeight: "400" | "600" | "700"; lineHeight: number }> = {
  title: { fontSize: fontSize.xxl, fontWeight: "700", lineHeight: 36 },
  heading: { fontSize: fontSize.xl, fontWeight: "700", lineHeight: 28 },
  body: { fontSize: fontSize.md, fontWeight: "400", lineHeight: 22 },
  caption: { fontSize: fontSize.sm, fontWeight: "400", lineHeight: 18 },
  label: { fontSize: fontSize.sm, fontWeight: "600", lineHeight: 18 }
};

export function Text({ variant = "body", tone = "default", center, style, ...rest }: TextProps) {
  const theme = useTheme();
  const colors: Record<Tone, string> = {
    default: theme.text,
    muted: theme.textMuted,
    primary: theme.primary,
    danger: theme.danger,
    success: theme.success,
    inverse: "#FFFFFF"
  };
  return <RNText {...rest} style={[sizes[variant], { color: colors[tone] }, center && { textAlign: "center" }, style]} />;
}
