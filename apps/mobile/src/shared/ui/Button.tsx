import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { hitSlop, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: ButtonVariant;
  loading?: boolean;
  disabled?: boolean;
  accessibilityHint?: string;
  /** Spoken label when the visible text is ambiguous (e.g. repeated row actions). */
  a11yLabel?: string;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({
  label,
  onPress,
  variant = "primary",
  loading = false,
  disabled = false,
  accessibilityHint,
  a11yLabel,
  style,
  testID
}: ButtonProps) {
  const theme = useTheme();
  const inactive = disabled || loading;
  const palette: Record<ButtonVariant, { bg: string; fg: string; border: string }> = {
    primary: { bg: theme.primary, fg: theme.onPrimary, border: theme.primary },
    secondary: { bg: theme.surface, fg: theme.text, border: theme.border },
    ghost: { bg: "transparent", fg: theme.primary, border: "transparent" },
    danger: { bg: theme.danger, fg: theme.onDanger, border: theme.danger }
  };
  const colors = palette[variant];
  return (
    <Pressable
      accessibilityHint={accessibilityHint}
      accessibilityLabel={a11yLabel ?? label}
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        { backgroundColor: colors.bg, borderColor: colors.border, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        style
      ]}
      testID={testID}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={colors.fg} size="small" /> : null}
        <Text style={{ color: colors.fg }} variant="label">
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: hitSlop + 4,
    paddingHorizontal: spacing.xl,
    borderRadius: radius.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm }
});
