import { useMemo } from "react";
import { StyleSheet } from "react-native";
import { useTheme, type Theme } from "./theme";

/**
 * Builds a StyleSheet from the current theme. Pass a module-level factory so the sheet is only rebuilt when the
 * theme (light/dark) changes.
 *
 *   const makeStyles = (t: Theme) => ({ box: { backgroundColor: t.colors.surface } });
 *   const styles = useThemedStyles(makeStyles);
 */
export function useThemedStyles<T extends StyleSheet.NamedStyles<T>>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => StyleSheet.create(factory(theme)), [factory, theme]);
}
