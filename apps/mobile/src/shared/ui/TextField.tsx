import React, { forwardRef, useState } from 'react';
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { fr } from '../../i18n/fr';
import { fontSize, radius, spacing, useTheme } from '../../theme';
import { Text } from './Text';

export type TextFieldProps = Omit<TextInputProps, 'style'> & {
  label: string;
  error?: string | null;
  hint?: string;
  /** Adds a show/hide toggle. */
  secure?: boolean;
  counter?: number;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, secure, counter, value, multiline, testID, ...rest },
  ref,
) {
  const { colors } = useTheme();
  const [hidden, setHidden] = useState(true);
  const [focused, setFocused] = useState(false);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted" style={styles.label}>
        {label}
      </Text>
      <View
        style={[
          styles.box,
          { backgroundColor: colors.surface, borderColor },
          multiline ? styles.multiline : null,
        ]}
      >
        <TextInput
          ref={ref}
          testID={testID}
          accessibilityLabel={label}
          accessibilityHint={hint}
          value={value}
          multiline={multiline}
          placeholderTextColor={colors.textMuted}
          secureTextEntry={secure ? hidden : rest.secureTextEntry}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          {...rest}
          style={[styles.input, { color: colors.text }, multiline ? styles.inputMulti : null]}
        />
        {secure ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? fr.auth.showPassword : fr.auth.hidePassword}
            testID={testID ? `${testID}-toggle` : undefined}
            onPress={() => setHidden((h) => !h)}
            hitSlop={8}
            style={styles.toggle}
          >
            <Ionicons
              name={hidden ? 'eye-outline' : 'eye-off-outline'}
              size={20}
              color={colors.textMuted}
            />
          </Pressable>
        ) : null}
      </View>
      <View style={styles.footer}>
        {error ? (
          <Text
            variant="caption"
            tone="danger"
            accessibilityRole="alert"
            testID={testID ? `${testID}-error` : undefined}
            style={styles.flex}
          >
            {error}
          </Text>
        ) : hint ? (
          <Text variant="caption" tone="muted" style={styles.flex}>
            {hint}
          </Text>
        ) : (
          <View style={styles.flex} />
        )}
        {counter !== undefined ? (
          <Text variant="caption" tone={value && value.length > counter ? 'danger' : 'muted'}>
            {(value ?? '').length}/{counter}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.md },
  label: { marginBottom: spacing.xs },
  box: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1.5,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    minHeight: 50,
  },
  multiline: { alignItems: 'flex-start', minHeight: 110 },
  input: {
    flex: 1,
    fontSize: fontSize.md,
    paddingVertical: spacing.md,
    outlineStyle: 'none',
  } as object,
  inputMulti: { minHeight: 90, textAlignVertical: 'top' },
  toggle: { paddingLeft: spacing.sm },
  footer: { flexDirection: 'row', marginTop: spacing.xs, minHeight: 16 },
  flex: { flex: 1 },
});
