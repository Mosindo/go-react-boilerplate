import React from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { MIN_TOUCH, fontSizes, radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = TextInputProps & { label: string; error?: string | null; hint?: string };

export const TextField = React.forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, style, multiline, ...rest },
  ref
) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="label" style={styles.label}>
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        multiline={multiline}
        style={[
          styles.input,
          {
            backgroundColor: colors.surface,
            color: colors.text,
            borderColor: error ? colors.danger : colors.border,
            minHeight: multiline ? 110 : MIN_TOUCH + 6,
            textAlignVertical: multiline ? "top" : "center"
          },
          style
        ]}
        {...rest}
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
  label: { marginLeft: 2 },
  input: {
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: fontSizes.md
  }
});
