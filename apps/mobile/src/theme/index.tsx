import React, { createContext, useContext, useMemo } from 'react';
import { StyleSheet, useColorScheme } from 'react-native';
import { darkPalette, lightPalette, type Palette } from './tokens';

export * from './tokens';

export type Theme = { colors: Palette; dark: boolean };

const ThemeContext = createContext<Theme>({ colors: lightPalette, dark: false });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => ({ colors: scheme === 'dark' ? darkPalette : lightPalette, dark: scheme === 'dark' }),
    [scheme],
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised themed StyleSheet. `factory` must be a stable (module level) function. */
export function useThemedStyles<T extends StyleSheet.NamedStyles<T>>(
  factory: (colors: Palette) => T,
): T {
  const { colors } = useTheme();
  return useMemo(() => StyleSheet.create(factory(colors)), [colors, factory]);
}
