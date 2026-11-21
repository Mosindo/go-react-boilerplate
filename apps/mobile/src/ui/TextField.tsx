import React, { forwardRef, useState } from "react";
import { Pressable, TextInput, View, type TextInputProps } from "react-native";
import { makeStyles, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type TextFieldProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  secureToggle?: boolean;
  counter?: number;
};

export const TextField = forwardRef<TextInput, TextFieldProps>(function TextField(
  { label, error, hint, secureToggle, counter, style, multiline, value, ...props },
  ref
) {
  const styles = useStyles();
  const theme = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(true);

  return (
    <View style={styles.wrapper}>
      <View style={styles.labelRow}>
        <Text variant="label" tone="muted">
          {label}
        </Text>
        {counter ? (
          <Text variant="caption" tone="subtle">
            {(value ?? "").length}/{counter}
          </Text>
        ) : null}
      </View>
      <View style={[styles.field, multiline ? styles.multiline : null, focused ? styles.focused : null, error ? styles.errored : null]}>
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          placeholderTextColor={theme.colors.textSubtle}
          style={[styles.input, multiline ? styles.inputMultiline : null, style]}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          secureTextEntry={secureToggle ? hidden : props.secureTextEntry}
          multiline={multiline}
          value={value}
          maxLength={counter}
          {...props}
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
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="subtle">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const useStyles = makeStyles((t) => ({
  wrapper: { gap: t.spacing.xs },
  labelRow: { flexDirection: "row", justifyContent: "space-between" },
  field: {
    minHeight: 52,
    borderRadius: t.radii.md,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface,
    paddingHorizontal: t.spacing.lg,
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.sm
  },
  multiline: { minHeight: 120, alignItems: "flex-start", paddingVertical: t.spacing.md },
  focused: { borderColor: t.colors.primary },
  errored: { borderColor: t.colors.danger },
  // The wrapper border shows focus, so the browser outline is redundant on web.
  input: { flex: 1, color: t.colors.text, fontSize: 16, paddingVertical: t.spacing.md, outlineWidth: 0 },
  inputMultiline: { textAlignVertical: "top", paddingVertical: 0, minHeight: 96 }
}));
