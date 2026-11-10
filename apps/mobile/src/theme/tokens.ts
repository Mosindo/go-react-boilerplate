import { Platform } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  surfaceRaised: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  textInverse: string;
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
  shadow: string;
};

// "Lueur": warm ivory and ink with an ember accent. Deliberately warm and
// quiet: no saturated gradients, a serif display face for names and titles.
export const light: Palette = {
  background: "#FBF6F1",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EBE3",
  surfaceRaised: "#FFFFFF",
  border: "#E8DDD3",
  borderStrong: "#D3C3B5",
  text: "#1F1A1D",
  textMuted: "#6B6064",
  textSubtle: "#8F8388",
  textInverse: "#FFFFFF",
  primary: "#C8553D",
  primaryPressed: "#AE4430",
  primarySoft: "#F8E2D9",
  onPrimary: "#FFFFFF",
  accent: "#2F5D50",
  accentSoft: "#DCEAE4",
  success: "#2F7D5B",
  successSoft: "#DFF1E8",
  danger: "#B4382A",
  dangerSoft: "#F9E0DC",
  overlay: "rgba(20, 15, 18, 0.55)",
  scrim: "rgba(20, 15, 18, 0.72)",
  shadow: "#2A1A12"
};

export const dark: Palette = {
  background: "#141113",
  surface: "#1D191C",
  surfaceAlt: "#272226",
  surfaceRaised: "#2A2428",
  border: "#352E33",
  borderStrong: "#4A4147",
  text: "#F6EFEA",
  textMuted: "#BBAFB4",
  textSubtle: "#8E8388",
  textInverse: "#1F1A1D",
  primary: "#EE7A56",
  primaryPressed: "#D8653F",
  primarySoft: "#3B241D",
  onPrimary: "#1A0F0B",
  accent: "#7FC2AC",
  accentSoft: "#1E3530",
  success: "#6CC79C",
  successSoft: "#1C3328",
  danger: "#F08573",
  dangerSoft: "#3D201B",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.78)",
  shadow: "#000000"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 24, xxl: 32, xxxl: 48 } as const;

export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;

export const fonts = {
  display: Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" }),
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" })
} as const;

export const typography = {
  display: { fontFamily: fonts.display, fontSize: 34, lineHeight: 40, letterSpacing: -0.5 },
  title: { fontFamily: fonts.display, fontSize: 26, lineHeight: 32, letterSpacing: -0.3 },
  heading: { fontFamily: fonts.body, fontSize: 18, lineHeight: 24, fontWeight: "600" as const },
  body: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23 },
  small: { fontFamily: fonts.body, fontSize: 14, lineHeight: 20 },
  caption: { fontFamily: fonts.body, fontSize: 12, lineHeight: 16, letterSpacing: 0.2 },
  label: { fontFamily: fonts.body, fontSize: 13, lineHeight: 18, fontWeight: "600" as const, letterSpacing: 0.3 }
} as const;

export type TypographyVariant = keyof typeof typography;
