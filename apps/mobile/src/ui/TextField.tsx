import React, { forwardRef } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radius, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  testID?: string;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, style, multiline, ...rest },
  ref,
) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="label" muted>
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        multiline={multiline}
        {...rest}
        style={[
          styles.input,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : colors.border,
          },
          multiline && styles.multiline,
          style,
        ]}
      />
      {error ? (
        <Text variant="caption" color={colors.danger} accessibilityRole="alert">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" muted>
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  input: {
    minHeight: 50,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16,
  },
  multiline: { minHeight: 110, textAlignVertical: "top" },
});
