import React from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, usePalette } from "../theme";
import { Text } from "./Text";

type Props = TextInputProps & { label: string; error?: string | null; hint?: string };

export function Input({ label, error, hint, style, ...rest }: Props) {
  const p = usePalette();
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={p.textMuted}
        {...rest}
        style={[
          styles.input,
          {
            color: p.text,
            backgroundColor: p.surface,
            borderColor: error ? p.danger : p.border
          },
          rest.multiline && styles.multiline,
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
}

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  input: {
    minHeight: 50,
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: 14,
    fontSize: 16
  },
  multiline: { minHeight: 110, paddingTop: 12, textAlignVertical: "top" }
});
