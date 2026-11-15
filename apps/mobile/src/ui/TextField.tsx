import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, type, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, secureToggle, style, multiline, ...rest },
  ref
) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(Boolean(secureToggle));

  return (
    <View style={styles.wrap}>
      <Text variant="label" color="textMuted">
        {label}
      </Text>
      <View
        style={[
          styles.field,
          multiline && styles.multiline,
          {
            backgroundColor: colors.surfaceAlt,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border
          }
        ]}
      >
        <TextInput
          ref={ref}
          placeholderTextColor={colors.textSubtle}
          selectionColor={colors.primary}
          accessibilityLabel={label}
          secureTextEntry={hidden}
          multiline={multiline}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          style={[type.body, styles.input, multiline && styles.inputMultiline, { color: colors.text }, style]}
          {...rest}
        />
        {secureToggle ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            onPress={() => setHidden((v) => !v)}
            hitSlop={10}
          >
            <Icon name={hidden ? "eye-outline" : "eye-off-outline"} size={20} color="textMuted" />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" color="textSubtle">
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
    borderRadius: radii.md,
    borderWidth: 1.5,
    paddingHorizontal: spacing.md,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  multiline: { alignItems: "flex-start", paddingVertical: spacing.sm },
  input: { flex: 1, paddingVertical: 12, outlineStyle: "none" } as object,
  inputMultiline: { minHeight: 110, textAlignVertical: "top", paddingVertical: 0 }
});
