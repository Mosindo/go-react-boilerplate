import React, { forwardRef, useState } from "react";
import { StyleSheet, TextInput, View, type StyleProp, type TextInputProps, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { controls, radii, spacing } from "./tokens";

export type InputProps = TextInputProps & {
  label?: string;
  hint?: string;
  error?: string | null;
  containerStyle?: StyleProp<ViewStyle>;
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { label, hint, error, containerStyle, multiline, style, onFocus, onBlur, ...props },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);

  return (
    <View style={[styles.container, containerStyle]}>
      {label ? (
        <Text tone="muted" variant="label" weight="semibold">
          {label}
        </Text>
      ) : null}
      <TextInput
        accessibilityLabel={props.accessibilityLabel ?? label}
        multiline={multiline}
        placeholderTextColor={colors.textSubtle}
        ref={ref}
        style={[
          styles.input,
          {
            minHeight: multiline ? controls.input.multiline : controls.input.md,
            backgroundColor: colors.surface,
            color: colors.text,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border,
            textAlignVertical: multiline ? "top" : "center"
          },
          style
        ]}
        onBlur={(event) => {
          setFocused(false);
          onBlur?.(event);
        }}
        onFocus={(event) => {
          setFocused(true);
          onFocus?.(event);
        }}
        {...props}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  container: { gap: spacing.xs },
  input: {
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    fontSize: 16
  }
});
