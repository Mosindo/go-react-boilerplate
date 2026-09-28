import React, { forwardRef } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { hitSlop, radius, spacing, typography, useTheme } from "../../theme";
import { Text } from "./Text";

export type InputProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

export const Input = forwardRef<TextInput, InputProps>(function Input({ label, error, hint, style, ...rest }, ref) {
  const theme = useTheme();
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        accessibilityHint={hint}
        placeholderTextColor={theme.textMuted}
        ref={ref}
        style={[
          styles.input,
          typography.body,
          {
            color: theme.text,
            backgroundColor: theme.surface,
            borderColor: error ? theme.danger : theme.border
          },
          style
        ]}
        {...rest}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="muted" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  input: {
    minHeight: hitSlop + 4,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  }
});
