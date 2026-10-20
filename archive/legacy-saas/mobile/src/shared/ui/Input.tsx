import React, { useState } from "react";
import {
  StyleSheet,
  TextInput as RNTextInput,
  type StyleProp,
  type TextInputProps as RNTextInputProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { FormField } from "./FormField";
import { colors, controls, radii, shadows, spacing, typography } from "./tokens";

export type InputProps = Omit<RNTextInputProps, "style"> & {
  label?: string;
  helperText?: string;
  error?: string | null;
  style?: StyleProp<TextStyle>;
  containerStyle?: StyleProp<ViewStyle>;
};

const styles = StyleSheet.create({
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radii.lg,
    backgroundColor: colors.backgroundElevated,
    color: colors.text,
    ...typography.body,
    minHeight: controls.input.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  multiline: {
    minHeight: controls.input.multiline,
    textAlignVertical: "top"
  },
  inputFocused: {
    borderColor: colors.primary,
    backgroundColor: colors.surface,
    ...shadows.focus
  },
  inputError: {
    borderColor: colors.danger
  },
  inputDisabled: {
    backgroundColor: colors.surfaceMuted,
    borderColor: colors.borderStrong,
    color: colors.textMuted
  }
});

export function Input({
  containerStyle,
  error,
  helperText,
  label,
  multiline,
  style,
  onBlur,
  onFocus,
  ...props
}: InputProps) {
  const [focused, setFocused] = useState(false);
  const disabled = props.editable === false;

  return (
    <FormField containerStyle={containerStyle} error={error} helperText={helperText} label={label}>
      <RNTextInput
        multiline={multiline}
        onBlur={(event) => {
          setFocused(false);
          onBlur?.(event);
        }}
        onFocus={(event) => {
          setFocused(true);
          onFocus?.(event);
        }}
        placeholderTextColor={colors.textMuted}
        selectionColor={colors.primary}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          focused && !disabled ? styles.inputFocused : null,
          error ? styles.inputError : null,
          disabled ? styles.inputDisabled : null,
          style
        ]}
        {...props}
      />
    </FormField>
  );
}
