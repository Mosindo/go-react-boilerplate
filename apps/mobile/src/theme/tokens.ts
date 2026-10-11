export interface ColorTokens {
  bg: string;
  bgOuter: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  textOnPrimary: string;
  primary: string;
  primarySoft: string;
  secondary: string;
  secondarySoft: string;
  textOnSecondary: string;
  danger: string;
  dangerSoft: string;
  success: string;
  successSoft: string;
  overlay: string;
  scrim: string;
  skeleton: string;
  photoPlaceholder: string;
  onPhoto: string;
}

export const lightColors: ColorTokens = {
  bg: '#FBF6EF',
  bgOuter: '#EFE6DA',
  surface: '#FFFFFF',
  surfaceAlt: '#F5EDE2',
  border: '#E6DACB',
  text: '#2A2320',
  textMuted: '#6F6158',
  textOnPrimary: '#FFFFFF',
  primary: '#C8472B',
  primarySoft: '#FBE4DC',
  secondary: '#5B2A5E',
  secondarySoft: '#EEDDEF',
  textOnSecondary: '#FFFFFF',
  danger: '#B3261E',
  dangerSoft: '#FAE1DE',
  success: '#2E7D5B',
  successSoft: '#DDF0E7',
  overlay: 'rgba(30, 20, 16, 0.5)',
  scrim: 'rgba(20, 12, 9, 0.55)',
  skeleton: '#EADFD1',
  photoPlaceholder: '#E3D3C3',
  onPhoto: '#FFFFFF',
};

export const darkColors: ColorTokens = {
  bg: '#1C1816',
  bgOuter: '#120F0E',
  surface: '#272120',
  surfaceAlt: '#322A28',
  border: '#46393A',
  text: '#F6EEE6',
  textMuted: '#B9AA9F',
  textOnPrimary: '#1A0E0A',
  primary: '#F0775A',
  primarySoft: '#4A2A22',
  secondary: '#C79BD0',
  secondarySoft: '#3D2A40',
  textOnSecondary: '#1F1021',
  danger: '#FF8A80',
  dangerSoft: '#4B2723',
  success: '#7FD1A8',
  successSoft: '#24392F',
  overlay: 'rgba(0, 0, 0, 0.6)',
  scrim: 'rgba(0, 0, 0, 0.6)',
  skeleton: '#3A302E',
  photoPlaceholder: '#3A302E',
  onPhoto: '#FFFFFF',
};

export const spacing = { xxs: 2, xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32, xxxl: 48 } as const;

export const radius = { sm: 8, md: 14, lg: 20, xl: 28, pill: 999 } as const;

/** Minimum interactive size (Apple HIG / Material). */
export const TOUCH_TARGET = 44;

/** Max width of the content column on wide screens (web / tablets). */
export const MAX_CONTENT_WIDTH = 480;

export const fontSizes = {
  caption: 13,
  small: 14,
  body: 16,
  lead: 18,
  heading: 22,
  title: 28,
  display: 36,
} as const;

export interface Theme {
  scheme: 'light' | 'dark';
  colors: ColorTokens;
  spacing: typeof spacing;
  radius: typeof radius;
  fontSizes: typeof fontSizes;
}

export const lightTheme: Theme = {
  scheme: 'light',
  colors: lightColors,
  spacing,
  radius,
  fontSizes,
};
export const darkTheme: Theme = { scheme: 'dark', colors: darkColors, spacing, radius, fontSizes };
