// Ember design tokens: warm, adult and calm. Raspberry accent on a soft
// paper-white (light) or deep plum (dark) base. All text pairs meet WCAG AA.
export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  accent: string;
  accentText: string;
  accentSoft: string;
  like: string;
  pass: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
};

export const lightPalette: Palette = {
  background: "#FBF7F4",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE8",
  border: "#E7DDD7",
  text: "#2B2230",
  textMuted: "#6B6072",
  accent: "#C2334F",
  accentText: "#FFFFFF",
  accentSoft: "#FBE7EB",
  like: "#1F9D6B",
  pass: "#6B6072",
  success: "#1F7A55",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  overlay: "rgba(43, 34, 48, 0.55)",
  scrim: "rgba(20, 12, 24, 0.62)"
};

export const darkPalette: Palette = {
  background: "#16111A",
  surface: "#211A27",
  surfaceAlt: "#2B2233",
  border: "#3A2F44",
  text: "#F4EEF6",
  textMuted: "#B2A5BA",
  accent: "#FF6B85",
  accentText: "#2B0A12",
  accentSoft: "#3D1F2A",
  like: "#4FD6A0",
  pass: "#B2A5BA",
  success: "#4FD6A0",
  danger: "#FF8A80",
  dangerSoft: "#3D1F1F",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.7)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;

export const fontSize = { caption: 13, body: 16, subtitle: 18, title: 24, display: 32 } as const;

export const MIN_TOUCH = 44;
