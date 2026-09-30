import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { useTheme } from "../ThemeProvider";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

export const TextField = forwardRef<TextInput, Props>(function TextField({ label, error, hint, style, multiline, ...props }, ref) {
  const { colors, radii, spacing, typography } = useTheme();
  const [focused, setFocused] = useState(false);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <TextInput
        accessibilityHint={hint}
        accessibilityLabel={label}
        multiline={multiline}
        onBlur={(e) => {
          setFocused(false);
          props.onBlur?.(e);
        }}
        onFocus={(e) => {
          setFocused(true);
          props.onFocus?.(e);
        }}
        placeholderTextColor={colors.textSubtle}
        ref={ref}
        style={[
          typography.body,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor,
            borderRadius: radii.md,
            paddingHorizontal: spacing.md,
            paddingVertical: spacing.sm,
            minHeight: multiline ? 110 : 52,
            textAlignVertical: multiline ? "top" : "center"
          },
          styles.input,
          style
        ]}
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
  wrap: { gap: 6 },
  input: { borderWidth: 1.5 }
});
