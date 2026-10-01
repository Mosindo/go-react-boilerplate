import React, { forwardRef } from "react";
import { StyleSheet, TextInput, type StyleProp, type TextInputProps, type ViewStyle } from "react-native";
import { useTheme } from "./theme";
import { controls, radii, spacing } from "./tokens";

export type InputProps = TextInputProps & {
  invalid?: boolean;
  multiline?: boolean;
  containerStyle?: StyleProp<ViewStyle>;
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { invalid = false, multiline = false, style, containerStyle: _containerStyle, ...props },
  ref
) {
  const { colors } = useTheme();
  return (
    <TextInput
      ref={ref}
      multiline={multiline}
      placeholderTextColor={colors.textSubtle}
      selectionColor={colors.primary}
      style={[
        styles.input,
        {
          backgroundColor: colors.surface,
          borderColor: invalid ? colors.danger : colors.border,
          color: colors.text
        },
        multiline && styles.multiline,
        style
      ]}
      {...props}
    />
  );
});

const styles = StyleSheet.create({
  input: {
    minHeight: controls.input.md,
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    fontSize: 16
  },
  multiline: {
    minHeight: controls.input.multiline,
    paddingTop: spacing.md,
    textAlignVertical: "top"
  }
});
