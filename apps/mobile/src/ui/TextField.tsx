import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { colors, radii, spacing, typography } from "../theme/tokens";
import { Text } from "./Text";

export type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  counter?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, counter, style, maxLength, value, onFocus, onBlur, ...rest },
  ref
) {
  const [focused, setFocused] = useState(false);
  return (
    <View style={styles.wrap}>
      <View style={styles.labelRow}>
        <Text variant="label" tone="muted">
          {label.toUpperCase()}
        </Text>
        {counter && maxLength ? (
          <Text variant="caption" tone="subtle">
            {(value ?? "").length}/{maxLength}
          </Text>
        ) : null}
      </View>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.accent}
        maxLength={maxLength}
        value={value}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        style={[styles.input, rest.multiline && styles.multiline, { borderColor: error ? colors.danger : focused ? colors.accent : colors.border }, style]}
        {...rest}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="subtle">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  labelRow: { flexDirection: "row", justifyContent: "space-between" },
  input: {
    ...typography.body,
    color: colors.text,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    minHeight: 52
  },
  multiline: { minHeight: 110, textAlignVertical: "top", paddingTop: spacing.md }
});
