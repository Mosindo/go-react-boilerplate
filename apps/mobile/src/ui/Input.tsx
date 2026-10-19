import React, { forwardRef } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { MIN_TOUCH, radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = TextInputProps & { label: string; error?: string | null; hint?: string };

export const Input = forwardRef<TextInput, Props>(function Input(
  { label, error, hint, style, ...rest },
  ref
) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="muted" bold>
        {label}
      </Text>
      <TextInput
        ref={ref}
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
          rest.multiline ? styles.multiline : null,
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
  wrap: { gap: spacing.xs },
  input: {
    minHeight: MIN_TOUCH + 4,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16
  },
  multiline: { minHeight: 110, textAlignVertical: "top" }
});
