import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: ButtonVariant;
  loading?: boolean;
  icon?: IconName;
  compact?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", loading = false, disabled, icon, compact = false, style, ...rest }: ButtonProps) {
  const { colors } = useTheme();
  const palette = {
    primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.surface, pressed: colors.surfaceMuted, fg: colors.text, border: colors.borderStrong },
    ghost: { bg: "transparent", pressed: colors.surfaceMuted, fg: colors.primary, border: "transparent" },
    danger: { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft }
  }[variant];
  const inactive = disabled || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        compact ? styles.compact : styles.regular,
        { backgroundColor: pressed ? palette.pressed : palette.bg, borderColor: palette.border, opacity: inactive && !loading ? 0.5 : 1 },
        style
      ]}
      {...rest}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <View style={styles.row}>
          {icon ? <Icon name={icon} size={18} color={palette.fg} /> : null}
          <Text variant="label" style={{ color: palette.fg, fontSize: compact ? 14 : 16 }}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

export function IconButton({
  icon,
  label,
  onPress,
  size = 44,
  tone = "default",
  disabled
}: {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "onPhoto";
  disabled?: boolean;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.iconButton,
        {
          width: size,
          height: size,
          backgroundColor: tone === "onPhoto" ? "rgba(0,0,0,0.35)" : pressed ? colors.surfaceMuted : "transparent"
        }
      ]}
    >
      <Icon name={icon} size={Math.round(size * 0.5)} color={tone === "onPhoto" ? colors.onPhoto : colors.text} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { borderRadius: radii.pill, alignItems: "center", justifyContent: "center", borderWidth: 1 },
  regular: { minHeight: 52, paddingHorizontal: spacing.xl },
  compact: { minHeight: 40, paddingHorizontal: spacing.md },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  iconButton: { borderRadius: radii.pill, alignItems: "center", justifyContent: "center" }
});
