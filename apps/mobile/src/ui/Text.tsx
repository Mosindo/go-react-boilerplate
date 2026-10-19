import React from "react";
import { Text as RNText, type TextProps, type TextStyle } from "react-native";
import { fontSize } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";

type Variant = "display" | "title" | "subtitle" | "body" | "caption";
type Tone = "default" | "muted" | "accent" | "danger" | "onAccent";

type Props = TextProps & { variant?: Variant; tone?: Tone; bold?: boolean };

const sizes: Record<Variant, TextStyle> = {
  display: { fontSize: fontSize.display, lineHeight: 38, fontWeight: "800" },
  title: { fontSize: fontSize.title, lineHeight: 30, fontWeight: "700" },
  subtitle: { fontSize: fontSize.subtitle, lineHeight: 24, fontWeight: "600" },
  body: { fontSize: fontSize.body, lineHeight: 22, fontWeight: "400" },
  caption: { fontSize: fontSize.caption, lineHeight: 18, fontWeight: "400" }
};

export function Text({ variant = "body", tone = "default", bold, style, ...rest }: Props) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    accent: colors.accent,
    danger: colors.danger,
    onAccent: colors.accentText
  }[tone];
  return <RNText {...rest} style={[sizes[variant], { color }, bold ? { fontWeight: "700" } : null, style]} />;
}
