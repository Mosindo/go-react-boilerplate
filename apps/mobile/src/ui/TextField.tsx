import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, typography, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

export type TextFieldProps = TextInputProps & {
  label: string;
  hint?: string;
  error?: string | null;
  secureToggle?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, hint, error, secureToggle, secureTextEntry, multiline, style, ...rest },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);
  const secure = secureToggle ? hidden : secureTextEntry;

  return (
    <View style={styles.wrap}>
      <Text color="textMuted" variant="label">
        {label}
      </Text>
      <View
        style={[
          styles.field,
          multiline ? styles.multiline : null,
          {
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : focused ? colors.accent : colors.border
          }
        ]}
      >
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          multiline={multiline}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          placeholderTextColor={colors.textSubtle}
          secureTextEntry={secure}
          style={[styles.input, { color: colors.text }, multiline ? styles.multilineInput : null, style]}
          {...rest}
        />
        {secureToggle ? (
          <Pressable
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => setHidden((v) => !v)}
          >
            <Icon color="textMuted" name={hidden ? "eye-outline" : "eye-off-outline"} size={20} />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text color="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text color="textSubtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  field: {
    minHeight: 52,
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  multiline: { alignItems: "flex-start", paddingVertical: spacing.sm },
  input: { flex: 1, ...typography.body, paddingVertical: spacing.sm, outlineWidth: 0 },
  multilineInput: { minHeight: 110, textAlignVertical: "top" }
});
