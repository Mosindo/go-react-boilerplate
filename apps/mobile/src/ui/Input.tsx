import React from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export type InputProps = TextInputProps & { label: string; error?: string | null };

export function Input({ label, error, style, ...rest }: InputProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="muted" style={styles.label}>
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        {...rest}
        style={[
          styles.input,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : colors.border
          },
          style
        ]}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityRole="alert" style={styles.error}>
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.md },
  label: { marginBottom: spacing.xs, marginLeft: spacing.xs },
  input: {
    minHeight: 50,
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16
  },
  error: { marginTop: spacing.xs, marginLeft: spacing.xs }
});
