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
    sm: 44,
    md: 50,
    lg: 56
  },
  /** Minimum touch target (WCAG / Apple HIG). */
  minTarget: 44,
  input: {
    md: 52,
    multiline: 128
  },
  avatar: {
    sm: 40,
    md: 48,
    lg: 64,
    xl: 96
  }
} as const;

export const shadows = {
  card: {
    shadowColor: "#0b1220",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#0b1220",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#C2452D",
    shadowOpacity: 0.16,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 0 },
    elevation: 0
  }
} as const;

export type TypographyToken = keyof typeof typography;
