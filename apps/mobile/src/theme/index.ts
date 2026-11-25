import { Platform } from "react-native";

// "Lueur" visual identity: a warm, low-light palette (dark-first) with an
// ember accent and a soft gold highlight. Contrast of text tokens against
// `background` meets WCAG AA.
export const colors = {
  background: "#15111A",
  backgroundElevated: "#1C1722",
  surface: "#241D2B",
  surfaceHigh: "#2E2637",
  border: "#3A3145",
  borderStrong: "#4D4259",
  text: "#F7EFE8",
  textMuted: "#BDB1BC",
  textSubtle: "#948897",
  primary: "#EE7A5F",
  primaryPressed: "#D8664C",
  primarySoft: "rgba(238, 122, 95, 0.16)",
  onPrimary: "#1B1013",
  gold: "#E9B872",
  goldSoft: "rgba(233, 184, 114, 0.16)",
  success: "#7CCBA6",
  successSoft: "rgba(124, 203, 166, 0.16)",
  danger: "#F27272",
  dangerSoft: "rgba(242, 114, 114, 0.14)",
  overlay: "rgba(10, 7, 12, 0.72)",
  scrim: "rgba(10, 7, 12, 0.55)",
  white: "#FFFFFF"
} as const;

export const spacing = {
  xxs: 4,
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 28,
  xxl: 36
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
  display: { fontFamily: displayFamily, fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.5 },
  title: { fontFamily: displayFamily, fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.3 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" as const },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  small: { fontSize: 14, lineHeight: 20, fontWeight: "400" as const },
  caption: { fontSize: 12, lineHeight: 16, fontWeight: "500" as const },
  label: { fontSize: 13, lineHeight: 18, fontWeight: "600" as const, letterSpacing: 0.4 }
} as const;

export type TypographyVariant = keyof typeof typography;

export const shadow = {
  card: Platform.select({
    web: { boxShadow: "0 16px 40px rgba(0,0,0,0.35)" },
    default: {
      shadowColor: "#000",
      shadowOpacity: 0.35,
      shadowRadius: 24,
      shadowOffset: { width: 0, height: 14 },
      elevation: 8
    }
  })
} as const;

export const layout = {
  maxContentWidth: 520
} as const;
