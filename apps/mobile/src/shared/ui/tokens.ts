export const colors = {
  primary: "#b23a5b",
  primarySoft: "#fbe9ee",
  primaryBorder: "#f1c4d1",
  primaryForeground: "#ffffff",
  secondary: "#6b5b66",
  secondarySoft: "#f3ece9",
  accent: "#e98a5b",
  accentSoft: "#fdeee5",
  background: "#fbf7f4",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#f7f1ed",
  surfaceSubtle: "#f1e9e4",
  surfaceAccent: "#fbe9ee",
  border: "#ebe1db",
  borderStrong: "#d6c8c0",
  text: "#2a1f2d",
  textMuted: "#6f6270",
  textSubtle: "#a09399",
  success: "#2f7d57",
  successSoft: "#e8f5ee",
  successBorder: "#b6e0c8",
  warning: "#a35a12",
  warningSoft: "#fdf1de",
  warningBorder: "#f3d49a",
  danger: "#c8342b",
  dangerSoft: "#fdeceb",
  dangerBorder: "#f4c2be",
  like: "#2f7d57",
  pass: "#6f6270",
  inverse: "#ffffff",
  overlay: "rgba(42, 31, 45, 0.45)",
  scrim: "rgba(20, 12, 22, 0.62)"
} as const;

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
  body: {
    fontSize: 15,
    lineHeight: 24,
    letterSpacing: -0.1
  },
  label: {
    fontSize: 13,
    lineHeight: 18,
    letterSpacing: 0.1
  },
  title: {
    fontSize: fontSizes.xxl,
    lineHeight: 40,
    letterSpacing: -1
  },
  heading: {
    fontSize: 22,
    lineHeight: 28,
    letterSpacing: -0.5
  },
  caption: {
    fontSize: fontSizes.xs,
    lineHeight: 16,
    letterSpacing: 0
  },
  eyebrow: {
    fontSize: fontSizes.xxs,
    lineHeight: 16,
    letterSpacing: 1.3,
    textTransform: "uppercase" as const
  },
  button: {
    fontSize: 15,
    lineHeight: 20,
    letterSpacing: -0.2
  }
} as const;

export const controls = {
  button: {
    sm: 40,
    md: 50,
    lg: 56
  },
  input: {
    md: 52,
    multiline: 128
  },
  avatar: {
    sm: 40,
    md: 44,
    lg: 56
  }
} as const;

export const shadows = {
  card: {
    shadowColor: "#2a1f2d",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#2a1f2d",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#b23a5b",
    shadowOpacity: 0.16,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 0 },
    elevation: 0
  }
} as const;

export const ui = {
  colors,
  spacing,
  radii,
  fontSizes,
  fontWeights,
  typography,
  controls,
  shadows
} as const;

export type ColorToken = keyof typeof colors;
export type TypographyToken = keyof typeof typography;
