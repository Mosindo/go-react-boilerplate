export type ColorTokens = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  primarySoft: string;
  danger: string;
  onDanger: string;
  success: string;
  overlay: string;
  scrim: string;
  onScrim: string;
  like: string;
  pass: string;
  bubbleMine: string;
  onBubbleMine: string;
  bubbleTheirs: string;
};

export const lightColors: ColorTokens = {
  background: "#FBF5EE",
  surface: "#FFFFFF",
  surfaceAlt: "#F3E9DF",
  border: "#E4D6C8",
  text: "#2A1830",
  textMuted: "#6B5A6E",
  primary: "#B84630",
  onPrimary: "#FFFFFF",
  primarySoft: "#F8DDD3",
  danger: "#A3262F",
  onDanger: "#FFFFFF",
  success: "#2F7A55",
  overlay: "rgba(42, 24, 48, 0.55)",
  scrim: "rgba(42, 24, 48, 0.62)",
  onScrim: "#FFFFFF",
  like: "#2F7A55",
  pass: "#8A4F63",
  bubbleMine: "#B84630",
  onBubbleMine: "#FFFFFF",
  bubbleTheirs: "#F3E9DF"
};

export const darkColors: ColorTokens = {
  background: "#1B1220",
  surface: "#281B2E",
  surfaceAlt: "#33243A",
  border: "#473552",
  text: "#F6ECE4",
  textMuted: "#BBA9BE",
  primary: "#E8765E",
  onPrimary: "#1B1220",
  primarySoft: "#4A2A2C",
  danger: "#F08C93",
  onDanger: "#1B1220",
  success: "#6FCB9F",
  overlay: "rgba(0, 0, 0, 0.62)",
  scrim: "rgba(0, 0, 0, 0.66)",
  onScrim: "#FFFFFF",
  like: "#6FCB9F",
  pass: "#D79AB0",
  bubbleMine: "#E8765E",
  onBubbleMine: "#1B1220",
  bubbleTheirs: "#33243A"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const MIN_TOUCH_TARGET = 44;

export const fontSize = { caption: 13, body: 16, lead: 18, heading: 22, title: 32 } as const;
