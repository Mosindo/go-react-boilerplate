import { DarkTheme, DefaultTheme, type Theme as NavigationTheme } from "@react-navigation/native";
import type { NativeStackNavigationOptions } from "@react-navigation/native-stack";
import { useMemo } from "react";
import { useTheme } from "../theme/useTheme";

export function useNavigationTheme(): NavigationTheme {
  const { colors, isDark } = useTheme();
  return useMemo(() => {
    const base = isDark ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: colors.primary,
        background: colors.background,
        card: colors.backgroundElevated,
        text: colors.text,
        border: colors.border,
        notification: colors.primary
      }
    };
  }, [colors, isDark]);
}

export function useStackScreenOptions(): NativeStackNavigationOptions {
  const { colors } = useTheme();
  return useMemo(
    () => ({
      headerStyle: { backgroundColor: colors.background },
      headerTintColor: colors.primary,
      headerTitleStyle: { color: colors.text, fontWeight: "700" as const },
      headerShadowVisible: false,
      contentStyle: { backgroundColor: colors.background }
    }),
    [colors]
  );
}
