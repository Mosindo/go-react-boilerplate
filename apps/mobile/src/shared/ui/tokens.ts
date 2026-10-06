// Design tokens for "Lien": warm, calm, adult. Light palette is cream + terracotta,
// dark palette is warm charcoal. Components read colours through useTheme()
// (see theme.ts); the static `colors` export is the light palette, kept for
// the legacy Tamagui token file.

export const lightColors = {
  primary: "#C9482F",
  primarySoft: "#FBEAE5",
  primaryBorder: "#F0C5BA",
  primaryForeground: "#FFFFFF",
  secondary: "#6B5E57",
  secondarySoft: "#F3ECE6",
  background: "#FAF6F1",
  backgroundElevated: "#FFFFFF",
  surface: "#FFFFFF",
  surfaceMuted: "#F6F0EA",
  surfaceSubtle: "#EFE6DE",
  surfaceAccent: "#FBEAE5",
  border: "#E9DED4",
  borderStrong: "#D5C6B9",
  text: "#241C18",
  textMuted: "#6F625A",
  textSubtle: "#9A8C83",
  success: "#2E7D5B",
  successSoft: "#E6F4EC",
  successBorder: "#B5DFC8",
  warning: "#9A5B0F",
  warningSoft: "#FFF3DC",
  warningBorder: "#F1D49A",
  danger: "#B3261E",
  dangerSoft: "#FCEDEB",
  dangerBorder: "#F2BDB8",
  inverse: "#FFFFFF",
  overlay: "rgba(24, 16, 12, 0.45)",
  like: "#2E9E6B",
  pass: "#8A7C73"
} as const;

export type Palette = { [K in keyof typeof lightColors]: string };

export const darkColors: Palette = {
  primary: "#F0745D",
  primarySoft: "#3A211B",
  primaryBorder: "#5E342A",
  primaryForeground: "#1A0F0B",
  secondary: "#B9AAA0",
  secondarySoft: "#2A221E",
  background: "#171210",
  backgroundElevated: "#211A17",
  surface: "#211A17",
  surfaceMuted: "#2A2220",
  surfaceSubtle: "#352B27",
  surfaceAccent: "#3A211B",
  border: "#3A2F2A",
  borderStrong: "#54463F",
  text: "#F6EEE8",
  textMuted: "#B7A89E",
  textSubtle: "#8B7C73",
  success: "#6CCB9B",
  successSoft: "#16301F",
  successBorder: "#245A3B",
  warning: "#F0B65A",
  warningSoft: "#33260F",
  warningBorder: "#5C4318",
  danger: "#FF8A80",
  dangerSoft: "#3A1815",
  dangerBorder: "#6A2B26",
  inverse: "#FFFFFF",
  overlay: "rgba(0, 0, 0, 0.6)",
  like: "#4CC38A",
  pass: "#A99B91"
};

export const colors: Palette = lightColors;

export const spacing = {
  xxs: 4,
  xs: 6,
  sm: 10,
  md: 14,
  lg: 18,
  xl: 24,
  xxl: 32,
  xxxl: 40
} as const;

export const radii = {
  xs: 8,
  sm: 10,
  md: 14,
  lg: 20,
  xl: 28,
  pill: 999
} as const;

export const fontSizes = {
  xxs: 11,
  xs: 12,
  sm: 14,
  md: 16,
  lg: 20,
  xl: 28,
  xxl: 36
} as const;

export const fontWeights = {
  regular: "400",
  medium: "500",
  semibold: "600",
  bold: "700"
} as const;

export const typography = {
  body: { fontSize: 16, lineHeight: 23, letterSpacing: -0.1 },
  label: { fontSize: 13, lineHeight: 18, letterSpacing: 0.1 },
  title: { fontSize: 30, lineHeight: 36, letterSpacing: -0.8 },
  heading: { fontSize: 21, lineHeight: 27, letterSpacing: -0.4 },
  caption: { fontSize: fontSizes.xs, lineHeight: 16, letterSpacing: 0 },
  eyebrow: { fontSize: fontSizes.xxs, lineHeight: 16, letterSpacing: 1.3, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20, letterSpacing: -0.2 }
} as const;

export const controls = {
  button: { sm: 40, md: 50, lg: 56 },
  input: { md: 52, multiline: 120 },
  avatar: { sm: 40, md: 48, lg: 64 }
} as const;

export const shadows = {
  card: {
    shadowColor: "#2A160D",
    shadowOpacity: 0.08,
    shadowRadius: 20,
    shadowOffset: { width: 0, height: 8 },
    elevation: 3
  },
  floating: {
    shadowColor: "#2A160D",
    shadowOpacity: 0.16,
    shadowRadius: 28,
    shadowOffset: { width: 0, height: 12 },
    elevation: 8
  }
} as const;

export type TypographyToken = keyof typeof typography;
