/**
 * Lumen design tokens.
 *
 * Warm, adult and quiet: ivory paper + aubergine ink, with a single ember accent
 * reserved for "like" and primary actions. `colors` is the light palette (kept
 * as a static export for tooling); components read the active palette through
 * `useTheme()` so light and dark follow the system setting.
 */
export const lightColors = {
  primary: "#c2452d",
  primarySoft: "#f8e4dd",
  primaryBorder: "#ebc1b5",
  primaryForeground: "#ffffff",
  secondary: "#5b2a4e",
  secondarySoft: "#efe3ec",
  background: "#faf6f1",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#f4eee7",
  surfaceSubtle: "#ede5db",
  surfaceAccent: "#f7e9e4",
  border: "#e6ddd2",
  borderStrong: "#cfc3b6",
  text: "#231b22",
  textMuted: "#6b5f68",
  textSubtle: "#8f838b",
  success: "#2f7d5b",
  successSoft: "#e5f3ec",
  successBorder: "#b5dcc9",
  warning: "#9a5b00",
  warningSoft: "#fdf1da",
  warningBorder: "#f1d29a",
  danger: "#c0342b",
  dangerSoft: "#fceae8",
  dangerBorder: "#f1bdb8",
  inverse: "#ffffff",
  overlay: "rgba(35, 27, 34, 0.45)",
  pass: "#6b5f68"
} as const;

export type ColorToken = keyof typeof lightColors;
export type Palette = Record<ColorToken, string>;

export const darkColors: Palette = {
  primary: "#f0735a",
  primarySoft: "#3a1f1a",
  primaryBorder: "#6a3328",
  primaryForeground: "#1b0f0c",
  secondary: "#e3b3d6",
  secondarySoft: "#3a2636",
  background: "#17121a",
  backgroundElevated: "#211a26",
  surface: "#211a26",
  surfaceMuted: "#2a2230",
  surfaceSubtle: "#342a3b",
  surfaceAccent: "#3a222b",
  border: "#3a3040",
  borderStrong: "#52465a",
  text: "#f6eff3",
  textMuted: "#bcaeb7",
  textSubtle: "#8f8189",
  success: "#6fcf9f",
  successSoft: "#17302a",
  successBorder: "#2c5a49",
  warning: "#f0b45a",
  warningSoft: "#33270f",
  warningBorder: "#5e4720",
  danger: "#ff8a80",
  dangerSoft: "#3a1b1a",
  dangerBorder: "#6a312d",
  inverse: "#ffffff",
  overlay: "rgba(0, 0, 0, 0.6)",
  pass: "#bcaeb7"
};

export const colors = lightColors;

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
  body: { fontSize: 16, lineHeight: 24, letterSpacing: -0.1 },
  label: { fontSize: 13, lineHeight: 18, letterSpacing: 0.1 },
  title: { fontSize: fontSizes.xl, lineHeight: 34, letterSpacing: -0.8 },
  heading: { fontSize: 22, lineHeight: 28, letterSpacing: -0.4 },
  caption: { fontSize: fontSizes.xs, lineHeight: 16, letterSpacing: 0 },
  eyebrow: { fontSize: fontSizes.xxs, lineHeight: 16, letterSpacing: 1.3, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20, letterSpacing: -0.2 }
} as const;

export const controls = {
  button: { sm: 44, md: 52, lg: 58 },
  input: { md: 52, multiline: 120 },
  avatar: { sm: 40, md: 48, lg: 64 }
} as const;

export const shadows = {
  card: {
    shadowColor: "#231b22",
    shadowOpacity: 0.08,
    shadowRadius: 22,
    shadowOffset: { width: 0, height: 8 },
    elevation: 3
  },
  floating: {
    shadowColor: "#231b22",
    shadowOpacity: 0.16,
    shadowRadius: 28,
    shadowOffset: { width: 0, height: 12 },
    elevation: 8
  },
  focus: {
    shadowColor: "#c2452d",
    shadowOpacity: 0.2,
    shadowRadius: 10,
    shadowOffset: { width: 0, height: 0 },
    elevation: 0
  }
} as const;

export const ui = { colors, spacing, radii, fontSizes, fontWeights, typography, controls, shadows } as const;

export type TypographyToken = keyof typeof typography;
