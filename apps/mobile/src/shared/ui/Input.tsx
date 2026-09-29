import React, { forwardRef, useState } from "react";
import {
  TextInput as RNTextInput,
  type StyleProp,
  type TextInputProps as RNTextInputProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { FormField } from "./FormField";
import { type Theme, useTheme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";
import { controls } from "./tokens";

export type InputProps = Omit<RNTextInputProps, "style"> & {
  label?: string;
  helperText?: string;
  hint?: string;
  error?: string | null;
  style?: StyleProp<TextStyle>;
  containerStyle?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  input: {
    borderWidth: 1,
    borderColor: t.colors.borderStrong,
    borderRadius: t.radii.md,
    backgroundColor: t.colors.surface,
    color: t.colors.text,
    ...t.typography.body,
    fontSize: 16,
    minHeight: controls.input.md,
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.sm
  },
  multiline: { minHeight: controls.input.multiline, textAlignVertical: "top" as const, paddingTop: t.spacing.md },
  focused: { borderColor: t.colors.primary, borderWidth: 2 },
  error: { borderColor: t.colors.danger },
  disabled: { backgroundColor: t.colors.surfaceMuted, color: t.colors.textMuted }
});

export const Input = forwardRef<RNTextInput, InputProps>(function Input(
  { containerStyle, error, helperText, hint, label, multiline, style, onBlur, onFocus, accessibilityLabel, ...props },
  ref
) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const [focused, setFocused] = useState(false);
  const disabled = props.editable === false;

  return (
    <FormField containerStyle={containerStyle} error={error} helperText={helperText} hint={hint} label={label}>
      <RNTextInput
        accessibilityLabel={accessibilityLabel ?? label}
        maxFontSizeMultiplier={1.6}
        multiline={multiline}
        onBlur={(event) => {
          setFocused(false);
          onBlur?.(event);
        }}
        onFocus={(event) => {
          setFocused(true);
          onFocus?.(event);
        }}
        placeholderTextColor={theme.colors.textSubtle}
        ref={ref}
        selectionColor={theme.colors.primary}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          focused && !disabled ? styles.focused : null,
          error ? styles.error : null,
          disabled ? styles.disabled : null,
          style
        ]}
        {...props}
      />
    </FormField>
  );
});
