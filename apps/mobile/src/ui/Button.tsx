import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { radii, spacing, useTheme, type Palette } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type ButtonProps = {
  label: string;
  onPress?: () => void;
  variant?: ButtonVariant;
  icon?: IconName;
  loading?: boolean;
  disabled?: boolean;
  size?: "md" | "lg";
  style?: StyleProp<ViewStyle>;
  testID?: string;
  accessibilityHint?: string;
};

function colorsFor(variant: ButtonVariant, colors: Palette) {
  switch (variant) {
    case "secondary":
      return { bg: colors.surfaceMuted, pressed: colors.border, fg: colors.text, border: colors.border };
    case "ghost":
      return { bg: "transparent", pressed: colors.surfaceMuted, fg: colors.primary, border: "transparent" };
    case "danger":
      return { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft };
    default:
      return { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary };
  }
}

export function Button({
  label,
  onPress,
  variant = "primary",
  icon,
  loading,
  disabled,
  size = "lg",
  style,
  testID,
  accessibilityHint
}: ButtonProps) {
  const { colors } = useTheme();
  const c = colorsFor(variant, colors);
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityHint={accessibilityHint}
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        size === "lg" ? styles.lg : styles.md,
        { backgroundColor: pressed ? c.pressed : c.bg, borderColor: c.border },
        inactive ? styles.disabled : null,
        style
      ]}
      testID={testID}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={c.fg} size="small" /> : icon ? <Icon color={c.fg} name={icon} size={20} /> : null}
        <Text style={{ color: c.fg }} variant="bodyStrong">
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.xl
  },
  lg: { minHeight: 54 },
  md: { minHeight: 44, paddingHorizontal: spacing.md },
  content: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  disabled: { opacity: 0.5 }
});
