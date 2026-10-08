import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, type TextInputProps } from "react-native";
import { useTheme } from "../../theme";
import { controls, radii, spacing, typography } from "./tokens";

export type InputProps = TextInputProps & { invalid?: boolean };

export const Input = forwardRef<TextInput, InputProps>(function Input({ invalid, multiline, style, onBlur, onFocus, ...props }, ref) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  return (
    <TextInput
      ref={ref}
      multiline={multiline}
      placeholderTextColor={colors.textSubtle}
      selectionColor={colors.primary}
      onBlur={(event) => {
        setFocused(false);
        onBlur?.(event);
      }}
      onFocus={(event) => {
        setFocused(true);
        onFocus?.(event);
      }}
      style={[
        styles.input,
        typography.body,
        {
          color: colors.text,
          backgroundColor: colors.surface,
          borderColor: invalid ? colors.danger : focused ? colors.primary : colors.border,
          minHeight: multiline ? controls.input.multiline : controls.input.md
        },
        multiline ? styles.multiline : null,
        style
      ]}
      {...props}
    />
  );
});

const styles = StyleSheet.create({
  input: { borderWidth: 1.5, borderRadius: radii.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  multiline: { paddingTop: spacing.md, textAlignVertical: "top" }
});
