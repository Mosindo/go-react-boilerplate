import React, { useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { Text } from "./Text";
import { radii, spacing, useTheme } from "./theme";

type InputProps = Omit<TextInputProps, "style"> & {
  label: string;
  error?: string | null;
  hint?: string;
};

export function Input({ label, error, hint, multiline, ...props }: InputProps) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);

  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        multiline={multiline}
        placeholderTextColor={colors.textMuted}
        selectionColor={colors.primary}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border
          }
        ]}
        {...props}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  input: {
    minHeight: 52,
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16
  },
  multiline: { minHeight: 110, textAlignVertical: "top" }
});
