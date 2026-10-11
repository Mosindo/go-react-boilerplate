import type { ReactNode } from 'react';
import { View } from 'react-native';
import { MAX_CONTENT_WIDTH, useTheme } from '../theme';

/** Centres the app in a ~480px column on wide screens (web, tablets). */
export function AppFrame({ children }: { children: ReactNode }) {
  const { colors } = useTheme();
  return (
    <View style={{ flex: 1, alignItems: 'center', backgroundColor: colors.bgOuter }}>
      <View
        style={{
          flex: 1,
          width: '100%',
          maxWidth: MAX_CONTENT_WIDTH,
          backgroundColor: colors.bg,
          overflow: 'hidden',
        }}
      >
        {children}
      </View>
    </View>
  );
}
