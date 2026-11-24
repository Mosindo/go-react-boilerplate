import React, { forwardRef, useState } from "react";
import { Platform, Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, radii, spacing, type as typeScale } from "../theme/tokens";
import { AppText } from "./AppText";

export type TextFieldProps = TextInputProps & {
  label?: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
  counter?: number;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, secureToggle, counter, style, secureTextEntry, value, ...rest },
  ref
) {
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);
  const isSecure = secureToggle ? hidden : secureTextEntry;
  return (
    <View style={styles.wrapper}>
      {label ? (
        <View style={styles.labelRow}>
          <AppText variant="label" color={colors.textMuted}>
            {label}
          </AppText>
          {counter !== undefined ? (
            <AppText variant="caption" color={colors.textSubtle}>
              {(value ?? "").length}/{counter}
            </AppText>
          ) : null}
        </View>
      ) : null}
      <View style={[styles.field, focused && styles.focused, !!error && styles.errored, rest.multiline && styles.multiline]}>
        <TextInput
          ref={ref}
          placeholderTextColor={colors.textSubtle}
          selectionColor={colors.primary}
          accessibilityLabel={label}
          {...rest}
          value={value}
          secureTextEntry={isSecure}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          style={[styles.input, rest.multiline && styles.inputMultiline, style]}
        />
        {secureToggle ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            onPress={() => setHidden((h) => !h)}
            style={styles.toggle}
          >
            <Ionicons name={hidden ? "eye-outline" : "eye-off-outline"} size={20} color={colors.textMuted} />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <AppText variant="caption" color={colors.danger} accessibilityLiveRegion="polite">
          {error}
        </AppText>
      ) : hint ? (
        <AppText variant="caption" color={colors.textSubtle}>
          {hint}
        </AppText>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrapper: { gap: spacing.xs },
  labelRow: { flexDirection: "row", justifyContent: "space-between" },
  field: {
    minHeight: 52,
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    flexDirection: "row",
    alignItems: "center"
  },
  multiline: { alignItems: "flex-start" },
  focused: { borderColor: colors.primary },
  errored: { borderColor: colors.danger },
  input: {
    flex: 1,
    minWidth: 0,
    width: "100%",
    color: colors.text,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    ...typeScale.body,
    // The field draws its own focus ring; hide the browser outline on web.
    ...Platform.select({ web: { outlineWidth: 0 }, default: {} })
  },
  inputMultiline: { minHeight: 110, textAlignVertical: "top" },
  toggle: { paddingHorizontal: spacing.sm, paddingVertical: spacing.sm }
});
