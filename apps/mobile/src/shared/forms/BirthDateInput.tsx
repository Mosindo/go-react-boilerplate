import React, { useRef } from 'react';
import { StyleSheet, TextInput, View } from 'react-native';
import { fr } from '../../i18n/fr';
import { fontSize, radius, spacing, useTheme } from '../../theme';
import { Text } from '../ui';

export type BirthDateParts = { day: string; month: string; year: string };

const digits = (s: string) => s.replace(/\D/g, '');

/** Accessible day/month/year entry (works identically on native and web, no picker dependency). */
export function BirthDateInput({
  value,
  onChange,
  error,
  editable = true,
}: {
  value: BirthDateParts;
  onChange: (next: BirthDateParts) => void;
  error?: string | null;
  editable?: boolean;
}) {
  const { colors } = useTheme();
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  const field = (
    key: keyof BirthDateParts,
    label: string,
    placeholder: string,
    max: number,
    width: number,
    ref?: React.RefObject<TextInput | null>,
    next?: React.RefObject<TextInput | null>,
  ) => (
    <View style={{ width }}>
      <TextInput
        ref={ref}
        testID={`birth-${key}`}
        accessibilityLabel={label}
        value={value[key]}
        editable={editable}
        placeholder={placeholder}
        placeholderTextColor={colors.textMuted}
        keyboardType="number-pad"
        inputMode="numeric"
        maxLength={max}
        onChangeText={(t) => {
          const v = digits(t);
          onChange({ ...value, [key]: v });
          if (next && v.length === max) next.current?.focus();
        }}
        style={[
          styles.input,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : colors.border,
          },
        ]}
      />
    </View>
  );
  return (
    <View accessibilityRole="none" accessibilityLabel={fr.onboarding.birthDate}>
      <View style={styles.row}>
        {field('day', fr.onboarding.day, fr.onboarding.dayPlaceholder, 2, 70, undefined, monthRef)}
        <Text tone="muted" style={styles.sep}>
          /
        </Text>
        {field(
          'month',
          fr.onboarding.month,
          fr.onboarding.monthPlaceholder,
          2,
          70,
          monthRef,
          yearRef,
        )}
        <Text tone="muted" style={styles.sep}>
          /
        </Text>
        {field('year', fr.onboarding.year, fr.onboarding.yearPlaceholder, 4, 100, yearRef)}
      </View>
      {error ? (
        <Text
          variant="caption"
          tone="danger"
          accessibilityRole="alert"
          testID="birth-error"
          style={styles.msg}
        >
          {error}
        </Text>
      ) : (
        <Text variant="caption" tone="muted" style={styles.msg}>
          {fr.onboarding.birthDateHint}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center' },
  sep: { marginHorizontal: spacing.sm },
  input: {
    height: 50,
    borderWidth: 1.5,
    borderRadius: radius.md,
    textAlign: 'center',
    fontSize: fontSize.md,
    paddingHorizontal: spacing.sm,
  },
  msg: { marginTop: spacing.xs },
});
