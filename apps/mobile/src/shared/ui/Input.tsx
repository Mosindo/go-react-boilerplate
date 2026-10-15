import React, { forwardRef, useState } from "react";
import { TextInput, type StyleProp, type TextInputProps, type TextStyle } from "react-native";
import { useTheme } from "../../theme";
import { controls, radii, spacing } from "./tokens";

export type InputProps = TextInputProps & {
  invalid?: boolean;
  style?: StyleProp<TextStyle>;
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { invalid = false, multiline, style, onBlur, onFocus, ...props },
  ref
) {
  const palette = useTheme();
  const [focused, setFocused] = useState(false);

  return (
    <TextInput
      ref={ref}
      allowFontScaling
      maxFontSizeMultiplier={1.4}
      multiline={multiline}
      placeholderTextColor={palette.textSubtle}
      selectionColor={palette.primary}
      style={[
        {
          minHeight: multiline ? controls.input.multiline : controls.input.md,
          borderRadius: radii.md,
          borderWidth: 1.5,
          borderColor: invalid ? palette.danger : focused ? palette.brand : palette.border,
          backgroundColor: palette.surface,
          color: palette.text,
          paddingHorizontal: spacing.lg,
          paddingTop: multiline ? spacing.md : 0,
          paddingBottom: multiline ? spacing.md : 0,
          fontSize: 16,
          textAlignVertical: multiline ? "top" : "center"
        },
        style
      ]}
      onBlur={(e) => {
        setFocused(false);
        onBlur?.(e);
      }}
      onFocus={(e) => {
        setFocused(true);
        onFocus?.(e);
      }}
      {...props}
    />
  );
});
