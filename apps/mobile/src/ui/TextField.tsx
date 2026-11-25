import React, { useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing, typography } from "../theme";
import { Text } from "./Text";

type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  showCounter?: boolean;
};

export function TextField({
  label,
  error,
  hint,
  secureTextEntry,
  multiline,
  maxLength,
  showCounter,
  value,
  style,
  ...rest
}: TextFieldProps) {
  const [focused, setFocused] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const secure = secureTextEntry && !revealed;

  return (
    <View style={styles.wrapper}>
      <Text variant="label" color="textMuted">
        {label}
      </Text>
      <View style={[styles.field, multiline && styles.multiline, focused && styles.focused, !!error && styles.errored]}>
        <TextInput
          {...rest}
          value={value}
          accessibilityLabel={label}
          placeholderTextColor={colors.textSubtle}
          selectionColor={colors.primary}
          secureTextEntry={secure}
          multiline={multiline}
          maxLength={maxLength}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          style={[styles.input, multiline && styles.inputMultiline, style]}
        />
        {secureTextEntry ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={revealed ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            hitSlop={10}
            onPress={() => setRevealed((v) => !v)}
          >
            <Ionicons name={revealed ? "eye-off-outline" : "eye-outline"} size={20} color={colors.textSubtle} />
          </Pressable>
        ) : null}
      </View>
      <View style={styles.footer}>
        {error ? (
          <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : hint ? (
          <Text variant="caption" color="textSubtle">
            {hint}
          </Text>
        ) : (
          <View />
        )}
        {showCounter && maxLength ? (
          <Text variant="caption" color="textSubtle">
            {(value ?? "").length}/{maxLength}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    gap: spacing.xxs
  },
  field: {
    minHeight: 52,
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    paddingHorizontal: spacing.md,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  multiline: {
    alignItems: "flex-start",
    paddingVertical: spacing.sm
  },
  focused: {
    borderColor: colors.primary
  },
  errored: {
    borderColor: colors.danger
  },
  input: {
    ...typography.body,
    flex: 1,
    color: colors.text,
    paddingVertical: spacing.sm
  },
  inputMultiline: {
    minHeight: 110,
    textAlignVertical: "top",
    paddingVertical: 0
  },
  footer: {
    minHeight: 16,
    flexDirection: "row",
    justifyContent: "space-between"
  }
});
