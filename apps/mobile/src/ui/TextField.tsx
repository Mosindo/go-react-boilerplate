import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, spacing, type Palette } from "../theme/palette";
import { useStyles, useTheme } from "../theme/ThemeProvider";
import { Icon } from "./Icon";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, secureToggle, style, multiline, secureTextEntry, ...rest },
  ref
) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);
  const secure = secureToggle ? hidden : secureTextEntry;
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={[styles.box, multiline ? styles.multiline : null, focused ? styles.focused : null, error ? styles.errored : null]}>
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
          style={[styles.input, multiline ? styles.inputMultiline : null, style]}
          {...rest}
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

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    wrap: { gap: spacing.xxs + 2 },
    box: {
      minHeight: 52,
      borderRadius: radii.md,
      borderWidth: 1,
      borderColor: c.border,
      backgroundColor: c.surface,
      paddingHorizontal: spacing.md,
      flexDirection: "row",
      alignItems: "center",
      gap: spacing.xs
    },
    multiline: { alignItems: "flex-start", paddingVertical: spacing.sm },
    focused: { borderColor: c.primary },
    errored: { borderColor: c.danger },
    input: { flex: 1, color: c.text, fontSize: 16, paddingVertical: spacing.sm, outlineStyle: "none" } as never,
    inputMultiline: { minHeight: 110, textAlignVertical: "top", paddingVertical: 0 }
  });
