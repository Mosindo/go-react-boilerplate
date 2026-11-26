import { Platform, type TextStyle } from "react-native";

export const spacing = {
  xxs: 4,
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 24,
  xxl: 32,
  xxxl: 48
} as const;

export const radii = {
  sm: 10,
  md: 14,
  lg: 20,
  xl: 28,
  pill: 999
} as const;

const displayFamily = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const typography = {
  display: { fontFamily: displayFamily, fontSize: 34, lineHeight: 40, fontWeight: "700", letterSpacing: -0.5 },
  title: { fontFamily: displayFamily, fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.3 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "700", letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
  overline: { fontSize: 12, lineHeight: 16, fontWeight: "700", letterSpacing: 1.1, textTransform: "uppercase" }
} satisfies Record<string, TextStyle>;

export type TypographyVariant = keyof typeof typography;

export const layout = {
  maxContentWidth: 560,
  touchTarget: 48
} as const;
