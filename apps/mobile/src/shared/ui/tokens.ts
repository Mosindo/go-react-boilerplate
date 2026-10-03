export const colors = {
  primary: "#c2455a",
  primarySoft: "#fbe9ec",
  primaryBorder: "#f1c3cb",
  primaryForeground: "#ffffff",
  secondary: "#5b4a52",
  secondarySoft: "#f3ece8",
  accent: "#e8a35b",
  background: "#fbf7f4",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#f7f1ed",
  surfaceSubtle: "#efe7e2",
  surfaceAccent: "#fdf1f3",
  border: "#eadfd9",
  borderStrong: "#d6c7bf",
  text: "#2b1f25",
  textMuted: "#76646b",
  textSubtle: "#a5949b",
  success: "#2f7d5b",
  successSoft: "#e9f6ef",
  successBorder: "#b8e2cc",
  warning: "#a15c12",
  warningSoft: "#fff4e0",
  warningBorder: "#f6d9a3",
  danger: "#c0362c",
  dangerSoft: "#fdeeec",
  dangerBorder: "#f6c6c1",
  inverse: "#ffffff",
  scrim: "rgba(30, 16, 22, 0.55)",
  overlay: "rgba(30, 16, 22, 0.38)"
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
    shadowColor: "#2b1f25",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#2b1f25",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#c2455a",
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
