import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { useTheme } from "../theme";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  counter?: number;
};

export const TextField = forwardRef<TextInput, Props>(function TextField({ label, error, hint, counter, style, multiline, value, ...props }, ref) {
  const { colors, radii, spacing, typography } = useTheme();
  const [focused, setFocused] = useState(false);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;
  return (
    <View style={styles.wrap}>
      <View style={styles.labelRow}>
        <Text tone="muted" variant="label">
          {label}
        </Text>
        {counter !== undefined ? (
          <Text tone="subtle" variant="caption">
            {(value ?? "").length}/{counter}
          </Text>
        ) : null}
      </View>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.primary}
        value={value}
        multiline={multiline}
        onFocus={(e) => {
          setFocused(true);
          props.onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          props.onBlur?.(e);
        }}
        style={[
          typography.body,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor,
            borderRadius: radii.md,
            paddingHorizontal: spacing.md,
            paddingVertical: multiline ? spacing.sm : 0,
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
  labelRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  input: { borderWidth: 1 }
});
