import React, { useState } from "react";
import {
  TextInput as RNTextInput,
  type StyleProp,
  type TextInputProps as RNTextInputProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { useTheme } from "../../theme/useTheme";
import { FormField } from "./FormField";
import { controls, radii, shadows, spacing, typography } from "./tokens";

export type InputProps = Omit<RNTextInputProps, "style"> & {
  label?: string;
  helperText?: string;
  error?: string | null;
  style?: StyleProp<TextStyle>;
  containerStyle?: StyleProp<ViewStyle>;
};

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
  const { colors } = useTheme();
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
        placeholderTextColor={colors.textSubtle}
        selectionColor={colors.primary}
        style={[
          {
            borderWidth: 1,
            borderColor: colors.border,
            borderRadius: radii.md,
            backgroundColor: colors.backgroundElevated,
            color: colors.text,
            ...typography.body,
            minHeight: controls.input.md,
            paddingHorizontal: spacing.lg,
            paddingVertical: spacing.md
          },
          multiline ? { minHeight: controls.input.multiline, textAlignVertical: "top" } : null,
          focused && !disabled ? { borderColor: colors.primary, ...shadows.focus } : null,
          error ? { borderColor: colors.danger } : null,
          disabled ? { backgroundColor: colors.surfaceMuted, color: colors.textMuted } : null,
          style
        ]}
        {...props}
      />
    </FormField>
  );
}
