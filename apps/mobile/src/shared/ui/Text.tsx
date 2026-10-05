import { Text as RNText, type TextProps as RNTextProps } from "react-native";
import { useTheme, fontSizes, type Colors } from "../../theme";

export type TextVariant = "hero" | "title" | "heading" | "body" | "bodyStrong" | "caption" | "label";
export type TextTone = "default" | "muted" | "primary" | "danger" | "success" | "onPrimary" | "onImage";

const variants = {
  hero: { fontSize: fontSizes.hero, lineHeight: 44, fontWeight: "800", letterSpacing: -1 },
  title: { fontSize: fontSizes.xxl, lineHeight: 34, fontWeight: "800", letterSpacing: -0.5 },
  heading: { fontSize: fontSizes.xl, lineHeight: 28, fontWeight: "700", letterSpacing: -0.3 },
  body: { fontSize: fontSizes.md, lineHeight: 23, fontWeight: "400" },
  bodyStrong: { fontSize: fontSizes.md, lineHeight: 23, fontWeight: "600" },
  caption: { fontSize: fontSizes.sm, lineHeight: 19, fontWeight: "400" },
  label: { fontSize: fontSizes.sm, lineHeight: 18, fontWeight: "600", letterSpacing: 0.2 }
} as const;

function toneColor(tone: TextTone, colors: Colors): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "primary":
      return colors.primary;
    case "danger":
      return colors.danger;
    case "success":
      return colors.success;
    case "onPrimary":
      return colors.onPrimary;
    case "onImage":
      return colors.onImage;
    default:
      return colors.text;
  }
}

export type TextProps = RNTextProps & { variant?: TextVariant; tone?: TextTone; center?: boolean };

export function Text({ variant = "body", tone = "default", center, style, ...props }: TextProps) {
  const { colors } = useTheme();
  return (
    <RNText
      accessibilityRole={variant === "hero" || variant === "title" || variant === "heading" ? "header" : undefined}
      {...props}
      style={[variants[variant], { color: toneColor(tone, colors) }, center && { textAlign: "center" }, style]}
    />
  );
}
