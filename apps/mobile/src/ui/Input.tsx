import { useState } from 'react';
import { Pressable, TextInput, View } from 'react-native';
import type { TextInputProps } from 'react-native';
import { fr } from '../i18n';
import { TOUCH_TARGET, useStyles, useTheme } from '../theme';
import { Text } from './Text';

interface InputProps extends Omit<TextInputProps, 'style'> {
  label: string;
  error?: string | null;
  helper?: string;
  /** Shows a character counter when set together with maxLength. */
  showCounter?: boolean;
}

export function Input({
  label,
  error,
  helper,
  showCounter,
  secureTextEntry,
  multiline,
  value,
  maxLength,
  onFocus,
  onBlur,
  ...rest
}: InputProps) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.xs },
    field: {
      flexDirection: 'row',
      alignItems: 'center',
      backgroundColor: t.colors.surface,
      borderWidth: 1.5,
      borderColor: t.colors.border,
      borderRadius: t.radius.md,
      paddingHorizontal: t.spacing.lg,
      minHeight: 52,
    },
    focused: { borderColor: t.colors.primary },
    errored: { borderColor: t.colors.danger },
    fieldMultiline: { alignItems: 'flex-start' },
    input: {
      flex: 1,
      fontSize: t.fontSizes.body,
      color: t.colors.text,
      paddingVertical: t.spacing.sm,
      outlineWidth: 0,
    },
    inputMultiline: { paddingVertical: t.spacing.md, minHeight: 110, textAlignVertical: 'top' },
    toggle: {
      minHeight: TOUCH_TARGET,
      minWidth: TOUCH_TARGET,
      justifyContent: 'center',
      alignItems: 'flex-end',
    },
    meta: { flexDirection: 'row', justifyContent: 'space-between', gap: t.spacing.md },
  }));
  return (
    <View style={styles.wrap}>
      <Text variant="label">{label}</Text>
      <View
        style={[
          styles.field,
          multiline && styles.fieldMultiline,
          focused && styles.focused,
          !!error && styles.errored,
        ]}
      >
        <TextInput
          {...rest}
          value={value}
          maxLength={maxLength}
          multiline={multiline}
          secureTextEntry={secureTextEntry && !revealed}
          accessibilityLabel={label}
          accessibilityHint={helper}
          placeholderTextColor={colors.textMuted}
          selectionColor={colors.primary}
          onFocus={(e) => {
            setFocused(true);
            onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            onBlur?.(e);
          }}
          style={[styles.input, multiline && styles.inputMultiline]}
        />
        {secureTextEntry ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={revealed ? fr.common.hidePassword : fr.common.showPassword}
            onPress={() => setRevealed((r) => !r)}
            style={styles.toggle}
          >
            <Text variant="small" color="primary" weight="600">
              {revealed ? fr.common.hide : fr.common.show}
            </Text>
          </Pressable>
        ) : null}
      </View>
      {error || helper || (showCounter && maxLength) ? (
        <View style={styles.meta}>
          {error ? (
            <Text
              variant="caption"
              color="danger"
              accessibilityLiveRegion="polite"
              style={{ flex: 1 }}
            >
              {error}
            </Text>
          ) : (
            <Text variant="caption" color="textMuted" style={{ flex: 1 }}>
              {helper ?? ''}
            </Text>
          )}
          {showCounter && maxLength ? (
            <Text variant="caption" color="textMuted">
              {(value ?? '').length}/{maxLength}
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  );
}
