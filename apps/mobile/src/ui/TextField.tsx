import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { fonts, radii, spacing } from "../theme/tokens";
import { Icon } from "./Icon";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
  counter?: boolean;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, secureToggle, counter, multiline, maxLength, value, style, ...props },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);
  const borderColor = error ? colors.danger : focused ? colors.primary : colors.border;

  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted" nativeID={`${label}-label`}>
        {label}
      </Text>
      <View style={[styles.field, { borderColor, backgroundColor: colors.surface }, multiline ? styles.multiline : null]}>
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          accessibilityLabelledBy={`${label}-label`}
          placeholderTextColor={colors.textSubtle}
          selectionColor={colors.primary}
          multiline={multiline}
          maxLength={maxLength}
          value={value}
          secureTextEntry={secureToggle ? hidden : props.secureTextEntry}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          style={[
            styles.input,
            { color: colors.text, fontFamily: fonts.body },
            multiline ? { minHeight: 110, textAlignVertical: "top" } : null,
            style
          ]}
          {...props}
        />
        {secureToggle ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            hitSlop={10}
            onPress={() => setHidden((h) => !h)}
          >
            <Icon name={hidden ? "eye-outline" : "eye-off-outline"} size={20} color={colors.textMuted} />
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
        {counter && maxLength ? (
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
    minHeight: 52,
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  multiline: { alignItems: "flex-start", paddingVertical: spacing.sm },
  input: { flex: 1, fontSize: 16, paddingVertical: 12, outlineStyle: "none" } as object,
  footer: { flexDirection: "row", justifyContent: "space-between", minHeight: 17 }
});
