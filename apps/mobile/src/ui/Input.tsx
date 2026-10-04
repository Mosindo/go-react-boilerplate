import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { MIN_TOUCH, fontSizes, radii, spacing, useColors } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

export type InputProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  /** Adds a show/hide toggle (use with secureTextEntry). */
  passwordToggle?: boolean;
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { label, error, hint, passwordToggle, secureTextEntry, style, multiline, ...rest },
  ref
) {
  const colors = useColors();
  const [revealed, setRevealed] = useState(false);
  const hidden = !!secureTextEntry && !revealed;

  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="small" weight="medium">
        {label}
      </Text>
      <View
        style={[
          styles.field,
          { backgroundColor: colors.surface, borderColor: error ? colors.danger : colors.border },
          multiline && styles.multiline
        ]}
      >
        <TextInput
          ref={ref}
          accessibilityHint={hint}
          accessibilityLabel={label}
          multiline={multiline}
          placeholderTextColor={colors.textMuted}
          secureTextEntry={hidden}
          style={[styles.input, { color: colors.text }, multiline && styles.inputMultiline, style]}
          {...rest}
        />
        {passwordToggle ? (
          <Pressable
            accessibilityLabel={revealed ? "Hide password" : "Show password"}
            accessibilityRole="button"
            onPress={() => setRevealed((value) => !value)}
            style={styles.toggle}
          >
            <Icon color={colors.textMuted} name={revealed ? "eye-off-outline" : "eye-outline"} size={20} />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="small">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="muted" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xxs + 2 },
  field: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: radii.md,
    minHeight: MIN_TOUCH + 6
  },
  multiline: { alignItems: "flex-start" },
  input: {
    flex: 1,
    minHeight: MIN_TOUCH + 4,
    paddingHorizontal: spacing.md,
    fontSize: fontSizes.body
  },
  inputMultiline: { minHeight: 110, paddingTop: spacing.sm, textAlignVertical: "top" },
  toggle: { width: MIN_TOUCH, height: MIN_TOUCH, alignItems: "center", justifyContent: "center" }
});
