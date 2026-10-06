import React, { useState } from "react";
import {
  TextInput as RNTextInput,
  type StyleProp,
  type TextInputProps as RNTextInputProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { FormField } from "./FormField";
import { useTheme } from "./theme";
import { controls, radii, spacing, typography } from "./tokens";

export type InputProps = Omit<RNTextInputProps, "style"> & {
  label?: string;
  helperText?: string;
  error?: string | null;
  style?: StyleProp<TextStyle>;
  containerStyle?: StyleProp<ViewStyle>;
};

export function Input({ containerStyle, error, helperText, label, multiline, style, onBlur, onFocus, ...props }: InputProps) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);

  return (
    <FormField containerStyle={containerStyle} error={error} helperText={helperText} label={label}>
      <RNTextInput
        accessibilityLabel={props.accessibilityLabel ?? label}
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
          typography.body,
          {
            borderWidth: 1,
            borderRadius: radii.lg,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border,
            backgroundColor: colors.surface,
            color: colors.text,
            minHeight: multiline ? controls.input.multiline : controls.input.md,
            paddingHorizontal: spacing.lg,
            paddingVertical: spacing.md,
            textAlignVertical: multiline ? "top" : "center"
          },
          style
        ]}
        {...props}
      />
    </FormField>
  );
}
