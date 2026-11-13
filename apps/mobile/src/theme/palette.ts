export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  surfaceRaised: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  border: string;
  borderStrong: string;
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
  shadow: string;
};

// "Aimant" identity: warm terracotta on cream, deep plum ink, sage accents.
export const lightPalette: Palette = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceAlt: "#F4EDE7",
  surfaceRaised: "#FFFFFF",
  text: "#231A20",
  textMuted: "#6B5D65",
  textSubtle: "#9C8F96",
  border: "#EADFD7",
  borderStrong: "#D6C7BC",
  primary: "#C2533A",
  primaryPressed: "#A94530",
  primarySoft: "#F7E3DC",
  onPrimary: "#FFFFFF",
  accent: "#2F6153",
  accentSoft: "#DDEBE5",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  success: "#2F6153",
  overlay: "rgba(35, 26, 32, 0.45)",
  scrim: "rgba(20, 14, 18, 0.72)",
  shadow: "#3A2430"
};

export const darkPalette: Palette = {
  background: "#140F13",
  surface: "#1D171B",
  surfaceAlt: "#281F25",
  surfaceRaised: "#2C2329",
  text: "#F6EEE9",
  textMuted: "#B9AAB2",
  textSubtle: "#887A82",
  border: "#33282F",
  borderStrong: "#4A3C44",
  primary: "#E0795E",
  primaryPressed: "#C9654B",
  primarySoft: "#3B241E",
  onPrimary: "#1A0F0B",
  accent: "#8CC2AE",
  accentSoft: "#1F3A31",
  danger: "#F97066",
  dangerSoft: "#3B1A18",
  success: "#8CC2AE",
  overlay: "rgba(0, 0, 0, 0.55)",
  scrim: "rgba(0, 0, 0, 0.78)",
  shadow: "#000000"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 24, xxl: 32, xxxl: 48 } as const;

export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;
