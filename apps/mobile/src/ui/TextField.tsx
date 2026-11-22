import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

export type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, style, multiline, onFocus, onBlur, ...props },
  ref
) {
  const [focused, setFocused] = useState(false);
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        multiline={multiline}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.accent}
        style={[styles.input, multiline ? styles.multiline : null, focused ? styles.focused : null, error ? styles.invalid : null, style]}
        {...props}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.xs
  },
  input: {
    minHeight: 52,
    borderRadius: radii.md,
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.border,
    color: colors.text,
    fontSize: 16,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm
  },
  multiline: {
    minHeight: 112,
    textAlignVertical: "top"
  },
  focused: {
    borderColor: colors.accent
  },
  invalid: {
    borderColor: colors.danger
  }
});
