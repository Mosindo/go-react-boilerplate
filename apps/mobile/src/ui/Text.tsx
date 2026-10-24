import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { useTheme } from "../theme/theme";

type Variant = "display" | "title" | "heading" | "body" | "label" | "caption";

const variants: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800", letterSpacing: -0.5 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "800", letterSpacing: -0.3 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
};

type Props = TextProps & { variant?: Variant; muted?: boolean; color?: string; center?: boolean };

export function Text({ variant = "body", muted, color, center, style, ...rest }: Props) {
  const { colors } = useTheme();
  return (
    <RNText
      {...rest}
      style={[
        variants[variant],
        { color: color ?? (muted ? colors.textMuted : colors.text) },
        center && { textAlign: "center" },
        style,
      ]}
    />
  );
}
