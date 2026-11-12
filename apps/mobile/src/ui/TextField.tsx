import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, type as typeScale, useTheme } from "../theme";
import { Text } from "./Text";

export type TextFieldProps = TextInputProps & {
  label: string;
  hint?: string;
  error?: string | null;
  counter?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, hint, error, counter, multiline, maxLength, value, style, onFocus, onBlur, ...rest },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;

  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        accessibilityHint={hint}
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.primary}
        multiline={multiline}
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
        style={[styles.input, typeScale.body, { color: colors.text, backgroundColor: colors.surface, borderColor }, multiline ? styles.multiline : null, style]}
        {...rest}
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
        {counter && maxLength ? (
          <Text variant="caption" tone="subtle">
            {(value ?? "").length}/{maxLength}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xxs },
  input: { minHeight: 52, borderWidth: 1.5, borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  multiline: { minHeight: 120, textAlignVertical: "top", paddingTop: spacing.sm },
  footer: { flexDirection: "row", justifyContent: "space-between", minHeight: 18 }
});
