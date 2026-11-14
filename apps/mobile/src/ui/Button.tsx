import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { radii, spacing, type Palette } from "../theme/palette";
import { useStyles, useTheme } from "../theme/ThemeProvider";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger" | "outline";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  size?: "md" | "lg" | "sm";
  loading?: boolean;
  icon?: IconName;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", size = "md", loading, icon, disabled, style, ...rest }: ButtonProps) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const isDisabled = disabled || loading;
  const fg = {
    primary: colors.onPrimary,
    secondary: colors.primary,
    ghost: colors.text,
    danger: colors.textInverse,
    outline: colors.text
  }[variant];
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        styles[size],
        styles[variant],
        pressed && !isDisabled ? styles[`${variant}Pressed` as const] : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...rest}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={fg} size="small" /> : icon ? <Icon color={fg} name={icon} size={18} /> : null}
        <Text variant="label" style={{ color: fg, fontSize: size === "sm" ? 14 : 16 }}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    base: { borderRadius: radii.pill, alignItems: "center", justifyContent: "center" },
    row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
    sm: { minHeight: 38, paddingHorizontal: spacing.md },
    md: { minHeight: 50, paddingHorizontal: spacing.xl },
    lg: { minHeight: 56, paddingHorizontal: spacing.xxl },
    primary: { backgroundColor: c.primary },
    primaryPressed: { backgroundColor: c.primaryPressed },
    secondary: { backgroundColor: c.primarySoft },
    secondaryPressed: { opacity: 0.8 },
    ghost: { backgroundColor: "transparent" },
    ghostPressed: { backgroundColor: c.surfaceMuted },
    outline: { backgroundColor: "transparent", borderWidth: 1, borderColor: c.borderStrong },
    outlinePressed: { backgroundColor: c.surfaceMuted },
    danger: { backgroundColor: c.danger },
    dangerPressed: { opacity: 0.85 },
    disabled: { opacity: 0.5 }
  });
