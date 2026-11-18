import { Platform } from "react-native";

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

const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const fonts = {
  display: serif,
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', sans-serif" })
} as const;

export const typography = {
  display: { fontFamily: fonts.display, fontSize: 34, lineHeight: 40, letterSpacing: -0.5 },
  title: { fontFamily: fonts.display, fontSize: 26, lineHeight: 32, letterSpacing: -0.3 },
  heading: { fontFamily: fonts.body, fontSize: 18, lineHeight: 24, fontWeight: "600" as const },
  body: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23 },
  bodyStrong: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontFamily: fonts.body, fontSize: 14, lineHeight: 19, fontWeight: "500" as const },
  caption: { fontFamily: fonts.body, fontSize: 13, lineHeight: 17 },
  overline: {
    fontFamily: fonts.body,
    fontSize: 12,
    lineHeight: 16,
    fontWeight: "600" as const,
    letterSpacing: 1,
    textTransform: "uppercase" as const
  }
} as const;

export type TypographyVariant = keyof typeof typography;

export const layout = {
  maxContentWidth: 560,
  touchTarget: 48
} as const;

export const motion = {
  fast: 140,
  normal: 220,
  slow: 320
} as const;
