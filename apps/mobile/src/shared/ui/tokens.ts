export const colors = {
  primary: "#d65a43",
  primarySoft: "#fdece7",
  primaryBorder: "#f4c3b8",
  primaryForeground: "#ffffff",
  secondary: "#6b5560",
  secondarySoft: "#f4ece8",
  background: "#fbf6f1",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#f7f0ea",
  surfaceSubtle: "#f0e6df",
  surfaceAccent: "#fdece7",
  border: "#eadfd6",
  borderStrong: "#d6c6ba",
  text: "#2b1b24",
  textMuted: "#6f5d66",
  textSubtle: "#9a8a91",
  success: "#1f7a4d",
  successSoft: "#e8f6ee",
  successBorder: "#b5e3c8",
  warning: "#a55a0a",
  warningSoft: "#fff4df",
  warningBorder: "#f6d9a0",
  danger: "#c4352a",
  dangerSoft: "#fdeeec",
  dangerBorder: "#f5c2bd",
  gold: "#e0a526",
  inverse: "#ffffff",
  overlay: "rgba(43, 27, 36, 0.5)"
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
    shadowColor: "#2b1b24",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#2b1b24",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#d65a43",
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
