import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { MIN_TOUCH, radii, spacing, useTheme } from "./theme";

type Variant = "primary" | "secondary" | "ghost" | "danger";

type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: Variant;
  loading?: boolean;
  disabled?: boolean;
  small?: boolean;
  testID?: string;
  accessibilityLabel?: string;
  style?: StyleProp<ViewStyle>;
};

export function Button({
  label,
  onPress,
  variant = "primary",
  loading = false,
  disabled = false,
  small = false,
  testID,
  accessibilityLabel,
  style
}: ButtonProps) {
  const { colors } = useTheme();
  const inactive = disabled || loading;

  const palette: Record<Variant, { bg: string; fg: string; border: string }> = {
    primary: { bg: colors.primary, fg: colors.primaryText, border: colors.primary },
    secondary: { bg: colors.surface, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    danger: { bg: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft }
  };
  const { bg, fg, border } = palette[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        small ? styles.small : null,
        { backgroundColor: bg, borderColor: border, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={fg} />
      ) : (
        <Text variant="label" style={{ color: fg, fontSize: small ? 14 : 16 }}>
          {label}
        </Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    paddingHorizontal: spacing.xl,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  small: { minHeight: MIN_TOUCH, paddingHorizontal: spacing.lg }
});
