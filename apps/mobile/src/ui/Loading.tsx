import { ActivityIndicator, View } from 'react-native';
import { fr } from '../i18n';
import { useTheme } from '../theme';

export function Loading() {
  const { colors } = useTheme();
  return (
    <View
      style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 }}
      accessibilityRole="progressbar"
      accessibilityLabel={fr.common.loading}
    >
      <ActivityIndicator size="large" color={colors.primary} />
    </View>
  );
}
