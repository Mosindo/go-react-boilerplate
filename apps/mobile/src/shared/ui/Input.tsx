import { Ionicons } from "@expo/vector-icons";
import { forwardRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { fontSizes, radii, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type InputProps = TextInputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

/** Labelled text field with inline error. Password fields get a show/hide toggle. */
export const Input = forwardRef<TextInput, InputProps>(function Input({ label, error, hint, secureTextEntry, style, ...props }, ref) {
  const { colors } = useTheme();
  const [revealed, setRevealed] = useState(false);
  const [focused, setFocused] = useState(false);
  const isPassword = !!secureTextEntry;

  return (
    <View style={styles.wrapper}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View
        style={[
          styles.field,
          { backgroundColor: colors.surface, borderColor: error ? colors.danger : focused ? colors.primary : colors.border }
        ]}
      >
        <TextInput
          ref={ref}
          accessibilityLabel={label}
          placeholderTextColor={colors.textMuted}
          secureTextEntry={isPassword && !revealed}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          style={[styles.input, { color: colors.text }, props.multiline && styles.multiline, style]}
          {...props}
        />
        {isPassword ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={revealed ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            onPress={() => setRevealed((v) => !v)}
            hitSlop={12}
          >
            <Ionicons name={revealed ? "eye-off-outline" : "eye-outline"} size={22} color={colors.textMuted} />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrapper: { gap: spacing.xs },
  field: { flexDirection: "row", alignItems: "center", borderWidth: 1.5, borderRadius: radii.md, paddingHorizontal: spacing.lg },
  input: { flex: 1, minHeight: 50, fontSize: fontSizes.md, paddingVertical: spacing.sm },
  multiline: { minHeight: 110, textAlignVertical: "top", paddingTop: spacing.md }
});
