import React, { useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radius, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = Omit<TextInputProps, "style"> & {
  label: string;
  error?: string | null;
  hint?: string;
  secure?: boolean;
};

export function TextField({ label, error, hint, secure, multiline, ...rest }: Props) {
  const theme = useTheme();
  const [hidden, setHidden] = useState(!!secure);
  const [focused, setFocused] = useState(false);

  return (
    <View style={styles.wrap}>
      <Text variant="label" muted>
        {label}
      </Text>
      <View
        style={[
          styles.field,
          {
            backgroundColor: theme.surface,
            borderColor: error ? theme.danger : focused ? theme.primary : theme.border
          }
        ]}
      >
        <TextInput
          accessibilityLabel={label}
          placeholderTextColor={theme.textMuted}
          {...rest}
          multiline={multiline}
          secureTextEntry={hidden}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          style={[styles.input, { color: theme.text }, multiline ? styles.multiline : null]}
        />
        {secure ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            onPress={() => setHidden((h) => !h)}
            hitSlop={8}
          >
            <Text variant="label" color={theme.primary}>
              {hidden ? "Afficher" : "Masquer"}
            </Text>
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text variant="caption" color={theme.danger} accessibilityRole="alert">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" muted>
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  field: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1.5,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    gap: spacing.sm
  },
  input: {
    flex: 1,
    minHeight: 50,
    fontSize: 16,
    paddingVertical: spacing.md,
    outlineStyle: "none"
  } as object,
  multiline: { minHeight: 96, textAlignVertical: "top" }
});
