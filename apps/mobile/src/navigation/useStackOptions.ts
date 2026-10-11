import type { NativeStackNavigationOptions } from '@react-navigation/native-stack';
import { useTheme } from '../theme';

export function useStackOptions(): NativeStackNavigationOptions {
  const { colors } = useTheme();
  return {
    headerStyle: { backgroundColor: colors.bg },
    headerTintColor: colors.text,
    headerTitleStyle: { fontWeight: '700' },
    headerShadowVisible: false,
    headerBackTitle: '',
    contentStyle: { backgroundColor: colors.bg },
  };
}
