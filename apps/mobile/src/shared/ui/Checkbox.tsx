import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { radius, spacing, useTheme } from '../../theme';
import { Text } from './Text';

export type CheckboxProps = {
  label: string;
  checked: boolean;
  onChange: (next: boolean) => void;
  error?: string | null;
  testID?: string;
};

function CheckboxImpl({ label, checked, onChange, error, testID }: CheckboxProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Pressable
        testID={testID}
        accessibilityRole="checkbox"
        accessibilityLabel={label}
        accessibilityState={{ checked }}
        onPress={() => onChange(!checked)}
        style={styles.row}
      >
        <View
          style={[
            styles.box,
            {
              borderColor: error ? colors.danger : checked ? colors.primary : colors.border,
              backgroundColor: checked ? colors.primary : colors.surface,
            },
          ]}
        >
          {checked ? <Ionicons name="checkmark" size={16} color={colors.onPrimary} /> : null}
        </View>
        <Text variant="body" style={styles.label}>
          {label}
        </Text>
      </Pressable>
      {error ? (
        <Text
          variant="caption"
          tone="danger"
          accessibilityRole="alert"
          testID={testID ? `${testID}-error` : undefined}
        >
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.md },
  row: { flexDirection: 'row', alignItems: 'center', minHeight: 44 },
  box: {
    width: 24,
    height: 24,
    borderRadius: radius.sm / 2 + 2,
    borderWidth: 2,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: spacing.md,
  },
  label: { flex: 1 },
});

export const Checkbox = React.memo(CheckboxImpl);
