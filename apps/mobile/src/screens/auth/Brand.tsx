import React from 'react';
import { StyleSheet, View } from 'react-native';
import { fr } from '../../i18n/fr';
import { spacing, useTheme } from '../../theme';
import { Text } from '../../shared/ui';

export function Brand({ light, tagline = true }: { light?: boolean; tagline?: boolean }) {
  const { colors } = useTheme();
  const fg = light ? '#FBF6F0' : colors.text;
  return (
    <View
      style={styles.wrap}
      accessible
      accessibilityRole="header"
      accessibilityLabel={`${fr.app.name}, ${fr.app.tagline}`}
    >
      <View style={styles.row}>
        <Text variant="display" style={{ color: fg, fontSize: 40, lineHeight: 46 }}>
          {fr.app.name}
        </Text>
        <View style={[styles.dot, { backgroundColor: light ? '#FF7A66' : colors.primary }]} />
      </View>
      {tagline ? (
        <Text tone="muted" style={{ color: light ? '#D9C7E0' : colors.textMuted }}>
          {fr.app.tagline}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.xl },
  row: { flexDirection: 'row', alignItems: 'flex-end' },
  dot: { width: 12, height: 12, borderRadius: 6, marginLeft: 4, marginBottom: 10 },
});
