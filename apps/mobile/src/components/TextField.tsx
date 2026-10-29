import React, { forwardRef, useState } from "react";
import { Pressable, StyleSheet, Text, TextInput, View, type TextInputProps } from "react-native";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  /** Adds a show/hide toggle for password fields. */
  revealable?: boolean;
};

export const TextField = forwardRef<TextInput, Props>(function TextField(
  { label, error, hint, revealable, secureTextEntry, multiline, style, ...rest },
  ref
) {
  const theme = useTheme();
  const [revealed, setRevealed] = useState(false);
  const [focused, setFocused] = useState(false);
  const hidden = Boolean(secureTextEntry) && !revealed;

  return (
    <View style={styles.wrap}>
      <Text style={[styles.label, { color: theme.textMuted }]}>{label}</Text>
      <View
        style={[
          styles.field,
          {
            backgroundColor: theme.surface,
            borderColor: error ? theme.danger : focused ? theme.primary : theme.border
          },
          multiline && styles.multiline
        ]}
      >
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          multiline={multiline}
          onBlur={() => setFocused(false)}
          onFocus={() => setFocused(true)}
          placeholderTextColor={theme.textMuted}
          secureTextEntry={hidden}
          selectionColor={theme.primary}
          style={[styles.input, { color: theme.text }, multiline && styles.inputMultiline, style]}
          {...rest}
        />
        {revealable ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={revealed ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            hitSlop={8}
            onPress={() => setRevealed((value) => !value)}
          >
            <Text style={[styles.reveal, { color: theme.primary }]}>
              {revealed ? "Masquer" : "Afficher"}
            </Text>
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" style={[styles.message, { color: theme.danger }]}>
          {error}
        </Text>
      ) : hint ? (
        <Text style={[styles.message, { color: theme.textMuted }]}>{hint}</Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { marginBottom: spacing.lg },
  label: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.6,
    marginBottom: spacing.xs,
    textTransform: "uppercase"
  },
  field: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg
  },
  multiline: { alignItems: "flex-start" },
  input: { flex: 1, minHeight: 52, fontSize: fontSizes.md },
  inputMultiline: { minHeight: 110, paddingTop: spacing.md, textAlignVertical: "top" },
  reveal: { fontSize: fontSizes.sm, fontWeight: "600", marginLeft: spacing.sm },
  message: { fontSize: fontSizes.xs, marginTop: spacing.xs }
});
