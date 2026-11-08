import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { radii, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

type Props = {
  title: string;
  onPress: () => void;
  variant?: Variant;
  loading?: boolean;
  disabled?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({
  title,
  onPress,
  variant = "primary",
  loading,
  disabled,
  icon,
  style,
  testID
}: Props) {
  const { colors } = useTheme();
  const inactive = disabled || loading;

  const palette = {
    primary: {
      bg: colors.primary,
      pressed: colors.primaryPressed,
      fg: colors.onPrimary,
      border: colors.primary
    },
    secondary: { bg: colors.surface, pressed: colors.surfaceMuted, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", pressed: colors.surfaceMuted, fg: colors.primary, border: "transparent" },
    danger: {
      bg: colors.dangerSoft,
      pressed: colors.dangerSoft,
      fg: colors.danger,
      border: colors.dangerSoft
    }
  }[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        {
          backgroundColor: pressed ? palette.pressed : palette.bg,
          borderColor: palette.border,
          opacity: inactive ? 0.55 : 1
        },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <View style={styles.row}>
          {icon ? <Ionicons name={icon} size={18} color={palette.fg} /> : null}
          <Text variant="label" style={{ color: palette.fg, fontSize: 16 }}>
            {title}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

type IconButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  background?: string;
  disabled?: boolean;
  badge?: number;
  testID?: string;
};

export function IconButton({
  icon,
  label,
  onPress,
  size = 44,
  color,
  background,
  disabled,
  badge,
  testID
}: IconButtonProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      hitSlop={8}
      style={({ pressed }) => [
        styles.icon,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: background ?? "transparent",
          opacity: disabled ? 0.4 : pressed ? 0.7 : 1
        }
      ]}
    >
      <Ionicons name={icon} size={size * 0.5} color={color ?? colors.text} />
      {badge && badge > 0 ? (
        <View style={[styles.badge, { backgroundColor: colors.primary }]}>
          <Text variant="caption" tone="onPrimary" style={styles.badgeText}>
            {badge > 9 ? "9+" : badge}
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  icon: { alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    top: -2,
    right: -2,
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    paddingHorizontal: 4,
    alignItems: "center",
    justifyContent: "center"
  },
  badgeText: { fontSize: 10, lineHeight: 12, fontWeight: "700" }
});
