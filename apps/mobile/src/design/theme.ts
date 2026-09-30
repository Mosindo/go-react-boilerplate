export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  surfaceRaised: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  border: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  accentSoft: string;
  danger: string;
  dangerSoft: string;
  success: string;
  overlay: string;
  scrim: string;
};

const light: Palette = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE6",
  surfaceRaised: "#FFFFFF",
  text: "#221C1F",
  textMuted: "#6B6064",
  textSubtle: "#978B90",
  border: "#E9E0D9",
  primary: "#B44D33",
  primaryPressed: "#983F29",
  primarySoft: "#F6E3DC",
  onPrimary: "#FFFFFF",
  accent: "#3F6B5B",
  accentSoft: "#E1EDE7",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  success: "#2F7A55",
  overlay: "rgba(20, 14, 17, 0.45)",
  scrim: "rgba(20, 14, 17, 0.62)"
};

const dark: Palette = {
  background: "#141013",
  surface: "#1D181B",
  surfaceMuted: "#2A2327",
  surfaceRaised: "#241E21",
  text: "#F5EDE7",
  textMuted: "#B7ABB0",
  textSubtle: "#8C8085",
  border: "#362D32",
  primary: "#E2765A",
  primaryPressed: "#C9644A",
  primarySoft: "#3B241E",
  onPrimary: "#1A0F0B",
  accent: "#7FB39E",
  accentSoft: "#1F2E28",
  danger: "#F97066",
  dangerSoft: "#3A1C1A",
  success: "#6FCF97",
  overlay: "rgba(0, 0, 0, 0.55)",
  scrim: "rgba(0, 0, 0, 0.7)"
};

export const palettes = { light, dark } as const;

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

export const typography = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.5 },
  heading: { fontSize: 20, lineHeight: 26, fontWeight: "600" as const, letterSpacing: -0.3 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const, letterSpacing: 0 },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const, letterSpacing: 0.1 },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" as const, letterSpacing: 0 },
  overline: { fontSize: 12, lineHeight: 16, fontWeight: "700" as const, letterSpacing: 1.2 }
} as const;

export type TypographyVariant = keyof typeof typography;

export type Theme = {
  dark: boolean;
  colors: Palette;
  spacing: typeof spacing;
  radii: typeof radii;
  typography: typeof typography;
};

export function buildTheme(dark: boolean): Theme {
  return { dark, colors: dark ? palettes.dark : palettes.light, spacing, radii, typography };
}
