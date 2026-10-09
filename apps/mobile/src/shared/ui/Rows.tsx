import React from 'react';
import { Pressable, StyleSheet, Switch, View } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { radius, spacing, useTheme } from '../../theme';
import { Text } from './Text';

export function Badge({ count, testID }: { count: number; testID?: string }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View
      testID={testID}
      style={[styles.badge, { backgroundColor: colors.primary }]}
      accessible={false}
    >
      <Text variant="caption" style={{ color: colors.onPrimary, fontWeight: '700' }}>
        {count > 99 ? '99+' : count}
      </Text>
    </View>
  );
}

export function ListRow({
  icon,
  label,
  value,
  onPress,
  destructive,
  badge,
  testID,
  hideChevron,
}: {
  icon?: keyof typeof Ionicons.glyphMap;
  label: string;
  value?: string;
  onPress: () => void;
  destructive?: boolean;
  badge?: number;
  testID?: string;
  hideChevron?: boolean;
}) {
  const { colors } = useTheme();
  const tint = destructive ? colors.danger : colors.text;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.row,
        { borderColor: colors.border },
        pressed ? { backgroundColor: colors.surfaceAlt } : null,
      ]}
    >
      {icon ? <Ionicons name={icon} size={22} color={tint} style={styles.rowIcon} /> : null}
      <Text variant="body" tone={destructive ? 'danger' : 'default'} style={styles.rowLabel}>
        {label}
      </Text>
      {value ? (
        <Text variant="caption" tone="muted" style={styles.rowValue}>
          {value}
        </Text>
      ) : null}
      {badge ? <Badge count={badge} testID={testID ? `${testID}-badge` : undefined} /> : null}
      {hideChevron ? null : <Ionicons name="chevron-forward" size={18} color={colors.textMuted} />}
    </Pressable>
  );
}

export function ToggleRow({
  label,
  help,
  value,
  onChange,
  disabled,
  testID,
}: {
  label: string;
  help?: string;
  value: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <View style={styles.flex}>
        <Text variant="body">{label}</Text>
        {help ? (
          <Text variant="caption" tone="muted">
            {help}
          </Text>
        ) : null}
      </View>
      <Switch
        testID={testID}
        accessibilityRole="switch"
        accessibilityLabel={label}
        value={value}
        disabled={disabled}
        onValueChange={onChange}
        trackColor={{ true: colors.primary, false: colors.border }}
        thumbColor="#FFFFFF"
      />
    </View>
  );
}

export type Choice<T extends string> = { value: T; label: string };

/** Single (radio) or multiple (checkbox) choice list rendered as selectable pills. */
export function ChoiceGroup<T extends string>({
  choices,
  selected,
  onToggle,
  mode,
  testIDPrefix,
  label,
}: {
  choices: readonly Choice<T>[];
  selected: readonly T[];
  onToggle: (value: T) => void;
  mode: 'single' | 'multiple';
  testIDPrefix: string;
  label: string;
}) {
  const { colors } = useTheme();
  return (
    <View
      accessibilityRole={mode === 'single' ? 'radiogroup' : 'none'}
      accessibilityLabel={label}
      style={styles.choices}
    >
      {choices.map((c) => {
        const on = selected.includes(c.value);
        return (
          <Pressable
            key={c.value}
            testID={`${testIDPrefix}-${c.value}`}
            accessibilityRole={mode === 'single' ? 'radio' : 'checkbox'}
            accessibilityLabel={c.label}
            accessibilityState={{ selected: on, checked: on }}
            onPress={() => onToggle(c.value)}
            style={[
              styles.choice,
              {
                borderColor: on ? colors.primary : colors.border,
                backgroundColor: on ? colors.primarySoft : colors.surface,
              },
            ]}
          >
            <Ionicons
              name={on ? 'checkmark-circle' : 'ellipse-outline'}
              size={22}
              color={on ? colors.primary : colors.textMuted}
              style={styles.rowIcon}
            />
            <Text variant="bodyStrong">{c.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

export function Stepper({
  label,
  value,
  min,
  max,
  step = 1,
  format,
  onChange,
  testID,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (v: number) => string;
  onChange: (v: number) => void;
  testID: string;
}) {
  const { colors } = useTheme();
  const btn = (dir: -1 | 1, icon: 'remove' | 'add', name: string) => {
    const next = Math.min(max, Math.max(min, value + dir * step));
    const disabled = next === value;
    return (
      <Pressable
        testID={`${testID}-${dir < 0 ? 'minus' : 'plus'}`}
        accessibilityRole="button"
        accessibilityLabel={`${name} : ${label}`}
        accessibilityState={{ disabled }}
        disabled={disabled}
        onPress={() => onChange(next)}
        style={[
          styles.stepBtn,
          { borderColor: colors.border, backgroundColor: colors.surface },
          disabled ? { opacity: 0.4 } : null,
        ]}
      >
        <Ionicons name={icon} size={20} color={colors.text} />
      </Pressable>
    );
  };
  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <Text variant="body" style={styles.flex}>
        {label}
      </Text>
      {btn(-1, 'remove', 'Diminuer')}
      <Text
        variant="bodyStrong"
        center
        style={styles.stepValue}
        testID={`${testID}-value`}
        accessibilityLiveRegion="polite"
      >
        {format ? format(value) : String(value)}
      </Text>
      {btn(1, 'add', 'Augmenter')}
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: 54,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  rowIcon: { marginRight: spacing.md },
  rowLabel: { flex: 1 },
  rowValue: { marginRight: spacing.sm },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: spacing.sm,
  },
  choices: { marginBottom: spacing.md },
  choice: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: 52,
    paddingHorizontal: spacing.lg,
    borderWidth: 1.5,
    borderRadius: radius.md,
    marginBottom: spacing.sm,
  },
  stepBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    borderWidth: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stepValue: { minWidth: 72 },
});
