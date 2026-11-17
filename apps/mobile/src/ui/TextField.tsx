import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { colors, radii, spacing, typography } from "../theme";
import { Text } from "./Text";

export type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  counter?: { max: number };
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, counter, style, multiline, value, onFocus, onBlur, ...props },
  ref
) {
  const [focused, setFocused] = useState(false);
  return (
    <View style={styles.root}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.accent}
        value={value}
        multiline={multiline}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        style={[styles.input, multiline ? styles.multiline : null, focused ? styles.focused : null, error ? styles.errored : null, style]}
        {...props}
      />
      <View style={styles.footer}>
        {error ? (
          <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : hint ? (
          <Text variant="caption" tone="subtle">
            {hint}
          </Text>
        ) : (
          <View />
        )}
        {counter ? (
          <Text variant="caption" tone={(value?.length ?? 0) > counter.max ? "danger" : "subtle"}>
            {value?.length ?? 0}/{counter.max}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  root: {
    gap: spacing.xs
  },
  input: {
    ...typography.body,
    color: colors.text,
    minHeight: 52,
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  multiline: {
    minHeight: 120,
    textAlignVertical: "top"
  },
  focused: {
    borderColor: colors.accent
  },
  errored: {
    borderColor: colors.danger
  },
  footer: {
    flexDirection: "row",
    justifyContent: "space-between",
    minHeight: 17
  }
});
