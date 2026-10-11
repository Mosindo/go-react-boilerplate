import { createContext, useContext, useMemo } from 'react';
import type { ReactNode } from 'react';
import { StyleSheet, useColorScheme } from 'react-native';
import { darkTheme, lightTheme } from './tokens';
import type { Theme } from './tokens';

const ThemeContext = createContext<Theme>(lightTheme);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme();
  const theme = scheme === 'dark' ? darkTheme : lightTheme;
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised themed StyleSheet: `const styles = useStyles((t) => ({ box: { ... } }))`. */
export function useStyles<T extends StyleSheet.NamedStyles<T>>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  // The factory is expected to be a pure function of the theme (declared inline per component).
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => StyleSheet.create(factory(theme)), [theme]);
}
