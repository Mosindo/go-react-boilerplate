import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, typography, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, secureToggle, style, multiline, secureTextEntry, ...props },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <View style={[styles.box, { borderColor, backgroundColor: colors.surface }, multiline ? styles.multiline : null]}>
        <TextInput
          accessibilityLabel={label}
          multiline={multiline}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          placeholderTextColor={colors.textSubtle}
          ref={ref}
          secureTextEntry={secureToggle ? hidden : secureTextEntry}
          style={[typography.body, styles.input, { color: colors.text }, multiline ? styles.multilineInput : null, style]}
          {...props}
        />
        {secureToggle ? (
          <Pressable
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => setHidden((v) => !v)}
          >
            <Icon color={colors.textMuted} name={hidden ? "eye-outline" : "eye-off-outline"} size={20} />
          </Pressable>
        ) : null}
      </View>
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
  wrap: { gap: 6 },
  box: {
    minHeight: 52,
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  input: { flex: 1, paddingVertical: spacing.sm, outlineStyle: "none" } as object,
  multiline: { alignItems: "flex-start", minHeight: 120 },
  multilineInput: { minHeight: 110, textAlignVertical: "top" }
});
