import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { usePalette } from "../theme";

type Variant = "title" | "heading" | "body" | "caption" | "label";
type Tone = "default" | "muted" | "primary" | "danger" | "inverse";

const variants: Record<Variant, TextStyle> = {
  title: { fontSize: 30, fontWeight: "800", letterSpacing: -0.5 },
  heading: { fontSize: 20, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23 },
  caption: { fontSize: 13, lineHeight: 18 },
  label: { fontSize: 14, fontWeight: "600" }
};

export function Text({
  variant = "body",
  tone = "default",
  style,
  ...rest
}: TextProps & { variant?: Variant; tone?: Tone }) {
  const p = usePalette();
  const color =
    tone === "muted"
      ? p.textMuted
      : tone === "primary"
        ? p.primary
        : tone === "danger"
          ? p.danger
          : tone === "inverse"
            ? "#FFFFFF"
            : p.text;
  return <RNText {...rest} style={[{ color }, variants[variant], style]} />;
}
