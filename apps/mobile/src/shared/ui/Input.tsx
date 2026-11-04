import React, { useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { Text } from "./Text";
import { MIN_TOUCH, radius, spacing, useColors } from "./theme";

type Props = Omit<TextInputProps, "style"> & {
  label: string;
  error?: string | null;
  hint?: string;
  multilineHeight?: number;
};

export function Input({ label, error, hint, secureTextEntry, multiline, multilineHeight = 110, ...rest }: Props) {
  const c = useColors();
  const [hidden, setHidden] = useState(true);
  const [focused, setFocused] = useState(false);
  const isSecret = Boolean(secureTextEntry);

  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">{label}</Text>
      <View
        style={[
          styles.field,
          { backgroundColor: c.surface, borderColor: error ? c.danger : focused ? c.primary : c.border },
          multiline ? { minHeight: multilineHeight, alignItems: "flex-start" } : null,
        ]}
      >
        <TextInput
          accessibilityLabel={label}
          placeholderTextColor={c.textMuted}
          multiline={multiline}
          secureTextEntry={isSecret && hidden}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          style={[styles.input, { color: c.text }, multiline ? { textAlignVertical: "top" } : null]}
          {...rest}
        />
        {isSecret ? (
          <Pressable accessibilityRole="button" accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"} onPress={() => setHidden((h) => !h)} style={styles.toggle}>
            <Text variant="label" tone="primary">{hidden ? "Afficher" : "Masquer"}</Text>
          </Pressable>
        ) : null}
      </View>
      {error ? <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">{error}</Text> : hint ? <Text variant="caption" tone="muted">{hint}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  field: { flexDirection: "row", alignItems: "center", borderWidth: 1.5, borderRadius: radius.md, paddingHorizontal: spacing.md, minHeight: MIN_TOUCH + 6 },
  input: { flex: 1, fontSize: 16, paddingVertical: spacing.sm },
  toggle: { minHeight: MIN_TOUCH, justifyContent: "center", paddingLeft: spacing.sm },
});
