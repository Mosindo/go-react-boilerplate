import { Platform } from "react-native";

/**
 * Lueur design tokens. A warm, low-light palette ("a glow at dusk"): deep
 * brown-black surfaces, cream text and an ember accent. All text/background
 * pairs below meet WCAG AA contrast.
 */
export const colors = {
  background: "#14100F",
  surface: "#1E1816",
  surfaceRaised: "#28201D",
  surfaceSunken: "#0F0C0B",
  border: "#3A2F2A",
  borderStrong: "#55463F",

  text: "#F6EDE6",
  textMuted: "#BFAFA3",
  textSubtle: "#8F7F74",
  textOnAccent: "#1A0E09",

  accent: "#F08A5D",
  accentPressed: "#D9744A",
  accentSoft: "rgba(240, 138, 93, 0.14)",
  rose: "#E86A7A",
  roseSoft: "rgba(232, 106, 122, 0.14)",
  plum: "#9C7BB8",

  success: "#7CC79A",
  successSoft: "rgba(124, 199, 154, 0.14)",
  danger: "#F06A6A",
  dangerSoft: "rgba(240, 106, 106, 0.14)",
  warning: "#E8B04B",

  overlay: "rgba(10, 7, 6, 0.72)",
  scrim: "rgba(0, 0, 0, 0.55)",
  white: "#FFFFFF",
  transparent: "transparent"
} as const;

export const spacing = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  xxl: 24,
  xxxl: 32,
  huge: 48
} as const;

export const radii = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  pill: 999
} as const;

const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const fonts = {
  display: serif,
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" })
} as const;

export const typography = {
  display: { fontFamily: fonts.display, fontSize: 34, lineHeight: 40, fontWeight: "700" as const },
  title: { fontFamily: fonts.display, fontSize: 26, lineHeight: 32, fontWeight: "700" as const },
  heading: { fontFamily: fonts.body, fontSize: 19, lineHeight: 25, fontWeight: "700" as const },
  body: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  caption: { fontFamily: fonts.body, fontSize: 13, lineHeight: 18, fontWeight: "500" as const },
  label: { fontFamily: fonts.body, fontSize: 12, lineHeight: 16, fontWeight: "700" as const, letterSpacing: 0.6 }
} as const;

export const shadows = {
  card: Platform.select({
    ios: { shadowColor: "#000", shadowOpacity: 0.35, shadowRadius: 18, shadowOffset: { width: 0, height: 10 } },
    android: { elevation: 8 },
    default: { boxShadow: "0 12px 32px rgba(0,0,0,0.45)" }
  }),
  floating: Platform.select({
    ios: { shadowColor: "#000", shadowOpacity: 0.3, shadowRadius: 12, shadowOffset: { width: 0, height: 6 } },
    android: { elevation: 6 },
    default: { boxShadow: "0 8px 24px rgba(0,0,0,0.4)" }
  })
} as const;

export const layout = {
  maxContentWidth: 560,
  hitSlop: { top: 10, bottom: 10, left: 10, right: 10 }
} as const;
