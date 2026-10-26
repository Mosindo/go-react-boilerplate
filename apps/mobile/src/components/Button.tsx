import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { radius, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type Props = {
  label: string;
  onPress?: () => void;
  variant?: ButtonVariant;
  loading?: boolean;
  disabled?: boolean;
  small?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({
  label,
  onPress,
  variant = "primary",
  loading,
  disabled,
  small,
  style,
  testID
}: Props) {
  const theme = useTheme();
  const inactive = disabled || loading;

  const palette = {
    primary: { bg: theme.primary, pressed: theme.primaryPressed, fg: theme.onPrimary, border: theme.primary },
    secondary: { bg: theme.surface, pressed: theme.surfaceAlt, fg: theme.text, border: theme.border },
    ghost: { bg: "transparent", pressed: theme.surfaceAlt, fg: theme.primary, border: "transparent" },
    danger: { bg: theme.danger, pressed: theme.danger, fg: theme.surface, border: theme.danger }
  }[variant];

  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        small ? styles.small : styles.regular,
        { backgroundColor: pressed ? palette.pressed : palette.bg, borderColor: palette.border },
        inactive ? styles.inactive : null,
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <Text variant="bodyStrong" color={palette.fg} style={small ? { fontSize: 14 } : null}>
          {label}
        </Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { borderRadius: radius.pill, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  regular: { minHeight: 50, paddingHorizontal: spacing.xl },
  small: { minHeight: 38, paddingHorizontal: spacing.lg },
  inactive: { opacity: 0.55 }
});
