import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { MIN_TOUCH_TARGET, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, secureTextEntry, style, multiline, ...rest },
  ref
) {
  const { colors } = useTheme();
  const [revealed, setRevealed] = useState(false);
  const [focused, setFocused] = useState(false);
  const isSecret = Boolean(secureTextEntry);

  return (
    <View style={styles.wrapper}>
      <Text color="textMuted" variant="caption">
        {label}
      </Text>
      <View
        style={[
          styles.field,
          {
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border
          },
          multiline ? styles.multiline : null
        ]}
      >
        <TextInput
          accessibilityLabel={label}
          multiline={multiline}
          placeholderTextColor={colors.textMuted}
          {...rest}
          onBlur={(event) => {
            setFocused(false);
            rest.onBlur?.(event);
          }}
          onFocus={(event) => {
            setFocused(true);
            rest.onFocus?.(event);
          }}
          ref={ref}
          secureTextEntry={isSecret && !revealed}
          selectionColor={colors.primary}
          style={[styles.input, { color: colors.text }, style]}
        />
        {isSecret ? (
          <Pressable
            accessibilityLabel={revealed ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            accessibilityRole="button"
            onPress={() => setRevealed((value) => !value)}
            style={styles.reveal}
          >
            <Text color="primary" variant="caption">
              {revealed ? "Masquer" : "Afficher"}
            </Text>
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" color="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text color="textMuted" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrapper: { gap: spacing.xs },
  field: {
    minHeight: MIN_TOUCH_TARGET + 8,
    borderWidth: 1,
    borderRadius: radius.md,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.lg
  },
  multiline: { alignItems: "flex-start", paddingVertical: spacing.sm },
  input: { flex: 1, fontSize: 16, paddingVertical: spacing.sm, minHeight: MIN_TOUCH_TARGET },
  reveal: { minHeight: MIN_TOUCH_TARGET, justifyContent: "center", paddingLeft: spacing.sm }
});
