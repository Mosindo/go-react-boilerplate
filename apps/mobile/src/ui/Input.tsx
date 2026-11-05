import React, { forwardRef } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, useTheme } from "../theme";
import { Text } from "./Text";

export type InputProps = TextInputProps & { label: string; error?: string | null; hint?: string };

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { label, error, hint, style, multiline, ...rest },
  ref
) {
  const t = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="label" style={styles.label}>
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={t.textMuted}
        multiline={multiline}
        {...rest}
        style={[
          styles.input,
          multiline && styles.multiline,
          {
            backgroundColor: t.surface,
            color: t.text,
            borderColor: error ? t.danger : t.border
          },
          style
        ]}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  label: { marginLeft: 4 },
  input: {
    minHeight: 52,
    borderRadius: radii.md,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 12,
    fontSize: 16
  },
  multiline: { minHeight: 110, textAlignVertical: "top" }
});
