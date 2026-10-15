// "Alma" visual identity: warm paper tones, deep plum ink and a coral accent.
// Everything that depends on light/dark lives in a Palette; sizes are shared.

export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  brand: string;
  brandSoft: string;
  like: string;
  pass: string;
  success: string;
  successSoft: string;
  warning: string;
  warningSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  isDark: boolean;
};

export const lightPalette: Palette = {
  background: "#FAF6F1",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  border: "#E8DED3",
  borderStrong: "#CFC1B2",
  text: "#2A2230",
  textMuted: "#6F6473",
  textSubtle: "#74697A",
  primary: "#BC3A22",
  primaryPressed: "#9F2F1A",
  primarySoft: "#FBE6E0",
  onPrimary: "#FFFFFF",
  brand: "#5E2650",
  brandSoft: "#EFE1EC",
  like: "#BC3A22",
  pass: "#6F6473",
  success: "#237A5B",
  successSoft: "#E1F3EB",
  warning: "#9A5B13",
  warningSoft: "#FBEFD9",
  danger: "#B42318",
  dangerSoft: "#FDE9E7",
  overlay: "rgba(26, 18, 31, 0.55)",
  scrim: "rgba(26, 18, 31, 0.28)",
  isDark: false
};

export const darkPalette: Palette = {
  background: "#17121A",
  surface: "#221B27",
  surfaceMuted: "#2C2432",
  border: "#392F40",
  borderStrong: "#54475C",
  text: "#F6EEF1",
  textMuted: "#B6A8B7",
  textSubtle: "#9C8FA0",
  primary: "#F0654C",
  primaryPressed: "#D9503A",
  primarySoft: "#381D19",
  onPrimary: "#1B1017",
  brand: "#D9A8CB",
  brandSoft: "#3A2535",
  like: "#F0654C",
  pass: "#B6A8B7",
  success: "#5CC49B",
  successSoft: "#1B3A2F",
  warning: "#E5B16A",
  warningSoft: "#3D2E17",
  danger: "#FF8A7D",
  dangerSoft: "#42201D",
  overlay: "rgba(0, 0, 0, 0.62)",
  scrim: "rgba(0, 0, 0, 0.4)",
  isDark: true
};

export const spacing = {
  xxs: 4,
  xs: 6,
  sm: 10,
  md: 14,
  lg: 18,
  xl: 24,
  xxl: 32,
  xxxl: 44
} as const;

export const radii = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 22,
  xl: 30,
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
  body: { fontSize: 16, lineHeight: 23 },
  label: { fontSize: 14, lineHeight: 19 },
  title: { fontSize: fontSizes.xxl, lineHeight: 42, letterSpacing: -0.8 },
  heading: { fontSize: 23, lineHeight: 29, letterSpacing: -0.3 },
  subheading: { fontSize: 18, lineHeight: 24, letterSpacing: -0.1 },
  caption: { fontSize: fontSizes.xs, lineHeight: 16 },
  eyebrow: { fontSize: fontSizes.xxs, lineHeight: 16, letterSpacing: 1.4, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20 }
} as const;

export const controls = {
  button: { sm: 40, md: 52, lg: 58 },
  input: { md: 52, multiline: 120 },
  avatar: { sm: 40, md: 52, lg: 72, xl: 112 },
  minTouch: 44
} as const;

export type TypographyToken = keyof typeof typography;
