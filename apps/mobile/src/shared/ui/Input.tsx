import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, type StyleProp, type TextInputProps, type ViewStyle } from "react-native";
import { FormField } from "./FormField";
import { colors, controls, radii, spacing, typography } from "./tokens";

export type InputProps = TextInputProps & {
  label?: string;
  helperText?: string;
  error?: string | null;
  containerStyle?: StyleProp<ViewStyle>;
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { containerStyle, error, helperText, label, multiline, onBlur, onFocus, style, ...props },
  ref
) {
  const [focused, setFocused] = useState(false);
  return (
    <FormField error={error} helperText={helperText} label={label} style={containerStyle}>
      <TextInput
        accessibilityLabel={label}
        multiline={multiline}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        placeholderTextColor={colors.textSubtle}
        ref={ref}
        selectionColor={colors.primary}
        style={[styles.input, multiline ? styles.multiline : null, focused ? styles.focused : null, error ? styles.error : null, style]}
        {...props}
      />
    </FormField>
  );
});

const styles = StyleSheet.create({
  input: {
    minHeight: controls.input.md,
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surfaceMuted,
    color: colors.text,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    ...typography.body
  },
  multiline: {
    minHeight: controls.input.multiline,
    textAlignVertical: "top"
  },
  focused: {
    borderColor: colors.primary,
    backgroundColor: colors.surface
  },
  error: {
    borderColor: colors.danger
  }
});
