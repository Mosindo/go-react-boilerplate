import { createFont, createTamagui } from "tamagui";
import { themes, tokens } from "./src/theme";

const bodyFont = createFont({
  family: "System",
  size: {
    1: 12,
    2: 14,
    3: 16,
    4: 18,
    5: 20,
    6: 24,
    true: 16
  },
  lineHeight: {
    1: 16,
    2: 20,
    3: 22,
    4: 24,
    5: 28,
    6: 32,
    true: 22
  },
  weight: {
    4: "400",
    5: "500",
    6: "600",
    7: "700"
  },
  letterSpacing: {
    4: 0,
    5: 0,
    6: 0,
    7: 0
  }
});

const headingFont = createFont({
  family: "System",
  size: {
    1: 14,
    2: 16,
    3: 18,
    4: 20,
    5: 24,
    6: 28,
    true: 18
  },
  lineHeight: {
    1: 18,
    2: 22,
    3: 24,
    4: 28,
    5: 30,
    6: 34,
    true: 24
  },
  weight: {
    5: "500",
    6: "600",
    7: "700",
    8: "800"
  },
  letterSpacing: {
    5: 0,
    6: 0,
    7: 0,
    8: 0
  }
});

const config = createTamagui({
  tokens,
  themes,
  fonts: {
    body: bodyFont,
    heading: headingFont
  },
  defaultTheme: "light"
});

export type AppTamaguiConfig = typeof config;

declare module "tamagui" {
  interface TamaguiCustomConfig extends AppTamaguiConfig {}
}

export default config;
