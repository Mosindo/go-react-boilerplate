import React, { useState } from "react";
import { TextInput, View, type TextInputProps } from "react-native";
import { createStyles, useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = Omit<TextInputProps, "style"> & {
  label: string;
  error?: string | null;
  hint?: string;
  counter?: string;
};

const useStyles = createStyles(({ colors, radius, spacing, minTouch, fontFamily }) => ({
  wrap: { gap: spacing.xs },
  input: {
    minHeight: minTouch + 4,
    borderRadius: radius.md,
    borderWidth: 1.5,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    color: colors.text,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16,
    fontFamily
  },
  focused: { borderColor: colors.primary },
  invalid: { borderColor: colors.danger },
  multiline: { minHeight: 120, textAlignVertical: "top" }
}));

export function TextField({
  label,
  error,
  hint,
  counter,
  multiline,
  onBlur,
  onFocus,
  ...rest
}: Props) {
  const styles = useStyles();
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  return (
    <View style={styles.wrap}>
      <Text variant="label">{label}</Text>
      <TextInput
        accessibilityLabel={label}
        accessibilityHint={hint}
        multiline={multiline}
        placeholderTextColor={colors.textMuted}
        {...rest}
        onFocus={(event) => {
          setFocused(true);
          onFocus?.(event);
        }}
        onBlur={(event) => {
          setFocused(false);
          onBlur?.(event);
        }}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          focused ? styles.focused : null,
          error ? styles.invalid : null
        ]}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      ) : null}
      {counter ? (
        <Text variant="caption" tone="muted" style={{ alignSelf: "flex-end" }}>
          {counter}
        </Text>
      ) : null}
    </View>
  );
}
