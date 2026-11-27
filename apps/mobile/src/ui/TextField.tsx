import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing, typography } from "../theme/palette";
import { Icon } from "./Icon";
import { Text } from "./Text";

type TextFieldProps = TextInputProps & {
  label: string;
  hint?: string;
  error?: string | null;
  secure?: boolean;
  showCounter?: boolean;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, hint, error, secure, showCounter, multiline, maxLength, value, style, ...props },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;

  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={[styles.field, { borderColor, backgroundColor: colors.surface }, multiline ? styles.multiline : null]}>
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          placeholderTextColor={colors.textSubtle}
          secureTextEntry={secure && !revealed}
          multiline={multiline}
          maxLength={maxLength}
          value={value}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          style={[
            typography.body,
            styles.input,
            { color: colors.text },
            multiline ? { minHeight: 110, textAlignVertical: "top" } : null,
            style
          ]}
          {...props}
        />
        {secure ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={revealed ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            onPress={() => setRevealed((v) => !v)}
            hitSlop={10}
            style={styles.reveal}
          >
            <Icon name={revealed ? "eye-off-outline" : "eye-outline"} size={20} color={colors.textMuted} />
          </Pressable>
        ) : null}
      </View>
      <View style={styles.footer}>
        {error ? (
          <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : hint ? (
          <Text variant="caption" tone="subtle">
            {hint}
          </Text>
        ) : (
          <View />
        )}
        {showCounter && maxLength ? (
          <Text variant="caption" tone="subtle">
            {(value ?? "").length}/{maxLength}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  field: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1.5,
    borderRadius: radii.md,
    minHeight: 52
  },
  multiline: { alignItems: "flex-start" },
  input: { flex: 1, paddingHorizontal: spacing.md, paddingVertical: 12 },
  reveal: { paddingHorizontal: spacing.sm },
  footer: { flexDirection: "row", justifyContent: "space-between", minHeight: 16 }
});
