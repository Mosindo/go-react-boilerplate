import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { radius, spacing, useTheme } from '../../theme';
import { Text } from './Text';

export type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
  /** Visual style for read-only chips on cards. */
  tone?: 'default' | 'overlay';
};

function ChipImpl({ label, selected, onPress, disabled, testID, tone = 'default' }: ChipProps) {
  const { colors } = useTheme();
  const bg =
    tone === 'overlay' ? 'rgba(255,255,255,0.22)' : selected ? colors.primary : colors.surfaceAlt;
  const fg = tone === 'overlay' ? '#FFFFFF' : selected ? colors.onPrimary : colors.text;
  const content = (
    <Text variant="label" style={{ color: fg }}>
      {label}
    </Text>
  );
  if (!onPress) {
    return <View style={[styles.chip, { backgroundColor: bg }]}>{content}</View>;
  }
  return (
    <Pressable
      testID={testID}
      accessibilityRole="checkbox"
      accessibilityLabel={label}
      accessibilityState={{ checked: !!selected, disabled: !!disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.chip,
        {
          backgroundColor: bg,
          borderColor: selected ? colors.primary : colors.border,
          borderWidth: 1,
        },
        pressed ? { opacity: 0.8 } : null,
        disabled ? { opacity: 0.45 } : null,
      ]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radius.pill,
    marginRight: spacing.sm,
    marginBottom: spacing.sm,
  },
});

export const Chip = React.memo(ChipImpl);
