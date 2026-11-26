export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  surfaceRaised: string;
  border: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  accentSoft: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  warning: string;
  overlay: string;
  scrim: string;
  shadow: string;
};

/** Warm ivory and ember: soft, adult and readable in daylight. */
export const lightPalette: Palette = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceMuted: "#F4EDE7",
  surfaceRaised: "#FFFFFF",
  border: "#E9DFD7",
  text: "#221B1E",
  textMuted: "#6F6368",
  textSubtle: "#9C8F95",
  primary: "#D4533D",
  primaryPressed: "#B94431",
  primarySoft: "#FBE3DC",
  onPrimary: "#FFFFFF",
  accent: "#74508A",
  accentSoft: "#EFE6F4",
  success: "#2E7D5B",
  successSoft: "#E3F3EA",
  danger: "#C0392B",
  dangerSoft: "#FBE5E2",
  warning: "#B7791F",
  overlay: "rgba(24, 16, 20, 0.55)",
  scrim: "rgba(20, 12, 16, 0.72)",
  shadow: "#3A2229"
};

/** Deep plum night with a glowing ember accent. */
export const darkPalette: Palette = {
  background: "#130F12",
  surface: "#1D171B",
  surfaceMuted: "#282026",
  surfaceRaised: "#241D22",
  border: "#362C33",
  text: "#F6EEE9",
  textMuted: "#B6A9B0",
  textSubtle: "#857880",
  primary: "#F07A5F",
  primaryPressed: "#D9664D",
  primarySoft: "#3B231F",
  onPrimary: "#1A0F0C",
  accent: "#BE94D1",
  accentSoft: "#2E2335",
  success: "#5BC495",
  successSoft: "#17291F",
  danger: "#F2746A",
  dangerSoft: "#3A1C1A",
  warning: "#E3B45B",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(8, 4, 6, 0.78)",
  shadow: "#000000"
};
