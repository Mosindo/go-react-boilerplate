import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

export type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  counter?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, counter, style, multiline, maxLength, value, onFocus, onBlur, ...props },
  ref
) {
  const theme = useTheme();
  const [focused, setFocused] = useState(false);
  const borderColor = error ? theme.colors.danger : focused ? theme.colors.primary : theme.colors.border;
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        maxLength={maxLength}
        multiline={multiline}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        placeholderTextColor={theme.colors.textSubtle}
        style={[
          theme.typography.body,
          styles.input,
          {
            color: theme.colors.text,
            backgroundColor: theme.colors.surface,
            borderColor,
            borderRadius: theme.radii.md,
            minHeight: multiline ? 120 : 52,
            textAlignVertical: multiline ? "top" : "center"
          },
          style
        ]}
        value={value}
        {...props}
      />
      <View style={styles.footer}>
        {error ? (
          <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
            {error}
          </Text>
        ) : hint ? (
          <Text tone="subtle" variant="caption">
            {hint}
          </Text>
        ) : (
          <View />
        )}
        {counter && maxLength ? (
          <Text tone="subtle" variant="caption">
            {(value ?? "").length}/{maxLength}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  input: { borderWidth: 1, paddingHorizontal: 16, paddingVertical: 12 },
  footer: { flexDirection: "row", justifyContent: "space-between", minHeight: 16 }
});
