/**
 * THEME / DARK MODE (read this before styling a screen)
 *
 * The app follows the system colour scheme (app.json userInterfaceStyle = automatic).
 *
 *   const { colors } = useTheme();                       // from "../theme"
 *   const styles = useMemo(() => createStyles(colors), [colors]);
 *   ...
 *   function createStyles(c: ColorPalette) { return StyleSheet.create({ root: { backgroundColor: c.background } }); }
 *
 * - Use `useTheme().colors` (same keys as the old static `colors`) for anything on screen.
 * - The static `colors` exported from "../shared/ui" is the LIGHT palette only: fine for
 *   non-visual constants, but do not use it for backgrounds/text or the screen breaks in dark mode.
 * - spacing, radii, fontSizes, fontWeights, typography, controls and shadows are scheme independent.
 * - All shared/ui + shared/layout + shared/feedback components already adapt automatically.
 * - Text drawn over photos: use `colors.onImage` (always white) and `colors.overlay` for scrims.
 * - Text on a `colors.primary` background: `colors.primaryForeground`.
 * - UI strings live in src/i18n/strings.ts (`strings.<area>.<key>`); add yours there.
 */
export { darkColors, lightColors, paletteFor } from "./palette";
export type { ColorPalette, ColorScheme } from "./palette";
export { useTheme } from "./useTheme";
export type { Theme } from "./useTheme";
export { controls, fontSizes, fontWeights, radii, shadows, spacing, typography } from "../shared/ui/tokens";
