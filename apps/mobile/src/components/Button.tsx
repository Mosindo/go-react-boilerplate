import React from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  title: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({
  title,
  onPress,
  variant = "primary",
  loading = false,
  disabled = false,
  style,
  testID
}: Props) {
  const theme = useTheme();
  const inactive = disabled || loading;

  const colors = {
    primary: {
      bg: theme.primary,
      pressed: theme.primaryPressed,
      fg: theme.textOnPrimary,
      border: theme.primary
    },
    secondary: {
      bg: theme.surface,
      pressed: theme.surfaceMuted,
      fg: theme.text,
      border: theme.border
    },
    ghost: {
      bg: "transparent",
      pressed: theme.surfaceMuted,
      fg: theme.primary,
      border: "transparent"
    },
    danger: {
      bg: theme.dangerSoft,
      pressed: theme.dangerSoft,
      fg: theme.danger,
      border: theme.danger
    }
  }[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        {
          backgroundColor: pressed ? colors.pressed : colors.bg,
          borderColor: colors.border,
          opacity: inactive && !loading ? 0.5 : 1
        },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={colors.fg} />
      ) : (
        <Text style={[styles.label, { color: colors.fg }]}>{title}</Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center"
  },
  label: { fontSize: fontSizes.md, fontWeight: "700", letterSpacing: 0.2 }
});
