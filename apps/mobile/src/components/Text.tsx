import React from "react";
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme";

export type TextVariant = "display" | "title" | "heading" | "body" | "bodyStrong" | "caption" | "label";

const variants: Record<TextVariant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800", letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.4 },
  heading: { fontSize: 19, lineHeight: 26, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
  label: { fontSize: 13, lineHeight: 18, fontWeight: "600", letterSpacing: 0.3 }
};

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  muted?: boolean;
  color?: string;
  center?: boolean;
};

export function Text({ variant = "body", muted, color, center, style, ...rest }: TextProps) {
  const theme = useTheme();
  return (
    <RNText
      accessibilityRole={variant === "display" || variant === "title" ? "header" : undefined}
      {...rest}
      style={[
        variants[variant],
        { color: color ?? (muted ? theme.textMuted : theme.text) },
        center ? { textAlign: "center" } : null,
        style
      ]}
    />
  );
}
