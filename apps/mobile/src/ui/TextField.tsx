import React, { forwardRef } from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { fontSizes, radii, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  helper?: string;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, helper, style, multiline, ...rest },
  ref
) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="textMuted" style={styles.label}>
        {label}
      </Text>
      <TextInput
        ref={ref}
        accessibilityLabel={label}
        placeholderTextColor={colors.textSubtle}
        multiline={multiline}
        {...rest}
        style={[
          styles.input,
          {
            color: colors.text,
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : colors.border,
            minHeight: multiline ? 112 : 52,
            textAlignVertical: multiline ? "top" : "center"
          },
          style
        ]}
      />
      {error ? (
        <Text variant="caption" tone="danger" style={styles.hint} accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : helper ? (
        <Text variant="caption" tone="textSubtle" style={styles.hint}>
          {helper}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.md },
  label: { marginBottom: spacing.xxs },
  input: {
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    fontSize: fontSizes.md
  },
  hint: { marginTop: spacing.xxs }
});
