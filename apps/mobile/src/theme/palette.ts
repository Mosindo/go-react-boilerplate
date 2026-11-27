import { Platform } from "react-native";

export type ColorScheme = "light" | "dark";

export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  surfaceRaised: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  accentSoft: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  pass: string;
};

// "Lueur" — a warm, candle-lit identity: cream paper, ink, terracotta ember
// and a touch of old gold. Contrast pairs are WCAG AA for body text.
export const palettes: Record<ColorScheme, Palette> = {
  light: {
    background: "#FBF7F2",
    surface: "#FFFFFF",
    surfaceMuted: "#F3ECE3",
    surfaceRaised: "#FFFFFF",
    border: "#E8DED2",
    borderStrong: "#D3C4B4",
    text: "#221A16",
    textMuted: "#6B5E56",
    textSubtle: "#9C8E84",
    primary: "#B9482F",
    primaryPressed: "#9E3B25",
    primarySoft: "#F8E3DB",
    onPrimary: "#FFFFFF",
    accent: "#B8862B",
    accentSoft: "#F6EAD0",
    success: "#2F6B4B",
    successSoft: "#E2F0E7",
    danger: "#B42318",
    dangerSoft: "#FDECEA",
    overlay: "rgba(20, 14, 11, 0.55)",
    scrim: "rgba(20, 14, 11, 0.72)",
    pass: "#4D423C"
  },
  dark: {
    background: "#14100E",
    surface: "#1E1815",
    surfaceMuted: "#2A221E",
    surfaceRaised: "#2F2622",
    border: "#3A302A",
    borderStrong: "#52453D",
    text: "#F5EDE6",
    textMuted: "#BFB1A7",
    textSubtle: "#8A7C72",
    primary: "#E8806A",
    primaryPressed: "#F0957F",
    primarySoft: "#3B2620",
    onPrimary: "#1A0F0B",
    accent: "#E0B565",
    accentSoft: "#3A2F1C",
    success: "#7CC29A",
    successSoft: "#1E3127",
    danger: "#F2867C",
    dangerSoft: "#3A1F1C",
    overlay: "rgba(0, 0, 0, 0.6)",
    scrim: "rgba(0, 0, 0, 0.78)",
    pass: "#D9CEC6"
  }
};

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
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" })
} as const;

export const typography = {
  display: { fontFamily: fonts.display, fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.5 },
  title: { fontFamily: fonts.display, fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.3 },
  heading: { fontFamily: fonts.body, fontSize: 19, lineHeight: 25, fontWeight: "700" as const, letterSpacing: -0.2 },
  body: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontFamily: fonts.body, fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontFamily: fonts.body, fontSize: 13, lineHeight: 18, fontWeight: "400" as const },
  overline: { fontFamily: fonts.body, fontSize: 12, lineHeight: 16, fontWeight: "700" as const, letterSpacing: 1.1 }
} as const;

export type TypographyVariant = keyof typeof typography;

export function shadow(scheme: ColorScheme, level: 1 | 2 = 1) {
  const opacity = scheme === "dark" ? 0.45 : level === 1 ? 0.08 : 0.16;
  return {
    shadowColor: "#1B120E",
    shadowOpacity: opacity,
    shadowRadius: level === 1 ? 14 : 28,
    shadowOffset: { width: 0, height: level === 1 ? 6 : 14 },
    elevation: level === 1 ? 3 : 10
  };
}
