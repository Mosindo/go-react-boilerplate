import React, { forwardRef } from "react";
import { StyleSheet, TextInput, type TextInputProps } from "react-native";
import { fontSize, MIN_TOUCH, radius, spacing, useTheme } from "../../theme";

export type InputProps = TextInputProps & { invalid?: boolean };

export const Input = forwardRef<TextInput, InputProps>(function Input({ invalid, style, multiline, ...rest }, ref) {
  const theme = useTheme();
  return (
    <TextInput
      ref={ref}
      placeholderTextColor={theme.textMuted}
      multiline={multiline}
      {...rest}
      style={[
        styles.input,
        {
          backgroundColor: theme.surface,
          borderColor: invalid ? theme.danger : theme.border,
          color: theme.text
        },
        multiline && styles.multiline,
        style
      ]}
    />
  );
});

const styles = StyleSheet.create({
  input: {
    minHeight: MIN_TOUCH + 4,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: fontSize.md
  },
  multiline: { minHeight: 96, textAlignVertical: "top" }
});
