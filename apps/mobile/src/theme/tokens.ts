import { Platform } from "react-native";

export type ColorScheme = "light" | "dark";

export type ThemeColors = {
  background: string;
  surface: string;
  surfaceAlt: string;
  surfaceRaised: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  textOnPrimary: string;
  textOnImage: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  like: string;
  pass: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  warning: string;
  overlay: string;
  scrim: string;
  shadow: string;
  skeleton: string;
};

// Warm, adult palette: ink, cream and terracotta. Contrast of text/primary
// colors on their backgrounds meets WCAG AA for body text.
const light: ThemeColors = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceAlt: "#F4EDE6",
  surfaceRaised: "#FFFFFF",
  border: "#E9E0D7",
  borderStrong: "#D3C6B9",
  text: "#1F1A17",
  textMuted: "#665C54",
  textSubtle: "#8F847A",
  textOnPrimary: "#FFFFFF",
  textOnImage: "#FFFFFF",
  primary: "#B4462F",
  primaryPressed: "#963A27",
  primarySoft: "#F6E3DC",
  like: "#B4462F",
  pass: "#4E4640",
  success: "#2F7D5B",
  successSoft: "#E2F1EA",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  warning: "#9A5B00",
  overlay: "rgba(24, 18, 15, 0.55)",
  scrim: "rgba(0, 0, 0, 0.6)",
  shadow: "#2A1A12",
  skeleton: "#EFE7DF"
};

const dark: ThemeColors = {
  background: "#14110F",
  surface: "#1D1916",
  surfaceAlt: "#27211D",
  surfaceRaised: "#2B2420",
  border: "#362E29",
  borderStrong: "#4A403A",
  text: "#F5EFE9",
  textMuted: "#BEB2A7",
  textSubtle: "#8C8178",
  textOnPrimary: "#1A0F0B",
  textOnImage: "#FFFFFF",
  primary: "#E57A63",
  primaryPressed: "#CC6650",
  primarySoft: "#3A2520",
  like: "#E57A63",
  pass: "#D9CFC6",
  success: "#6CC59C",
  successSoft: "#1C2E26",
  danger: "#F2867B",
  dangerSoft: "#3A1E1B",
  warning: "#E3A64B",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.65)",
  shadow: "#000000",
  skeleton: "#27211D"
};

export const palettes: Record<ColorScheme, ThemeColors> = { light, dark };

export const spacing = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  xxxl: 48
} as const;

export const radii = {
  sm: 8,
  md: 14,
  lg: 20,
  xl: 28,
  pill: 999
} as const;

const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const typography = {
  display: { fontFamily: serif, fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.5 },
  title: { fontFamily: serif, fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.3 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" as const, letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" as const },
  micro: { fontSize: 11, lineHeight: 14, fontWeight: "700" as const, letterSpacing: 0.8 }
} as const;

export type TypographyVariant = keyof typeof typography;

export type Theme = {
  scheme: ColorScheme;
  colors: ThemeColors;
  spacing: typeof spacing;
  radii: typeof radii;
  typography: typeof typography;
};

export function buildTheme(scheme: ColorScheme): Theme {
  return { scheme, colors: palettes[scheme], spacing, radii, typography };
}
