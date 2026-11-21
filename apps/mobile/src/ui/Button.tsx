import React from "react";
import { ActivityIndicator, Pressable, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { makeStyles, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: ButtonVariant;
  size?: "md" | "lg" | "sm";
  icon?: IconName;
  loading?: boolean;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", size = "md", icon, loading, fullWidth, disabled, style, ...props }: ButtonProps) {
  const styles = useStyles();
  const theme = useTheme();
  const isDisabled = disabled || loading;
  const fg =
    variant === "primary"
      ? theme.colors.textOnPrimary
      : variant === "danger"
        ? theme.colors.danger
        : variant === "secondary"
          ? theme.colors.text
          : theme.colors.primary;

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
        pressed && variant === "primary" ? styles.primaryPressed : null,
        pressed && variant !== "primary" ? styles.pressed : null,
        fullWidth ? styles.fullWidth : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={fg} size="small" /> : icon ? <Icon name={icon} size={18} color={fg} /> : null}
        <Text variant={size === "sm" ? "label" : "bodyStrong"} style={{ color: fg }}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const useStyles = makeStyles((t) => ({
  base: { borderRadius: t.radii.pill, alignItems: "center", justifyContent: "center" },
  sm: { minHeight: 38, paddingHorizontal: t.spacing.lg },
  md: { minHeight: 50, paddingHorizontal: t.spacing.xl },
  lg: { minHeight: 56, paddingHorizontal: t.spacing.xl },
  primary: { backgroundColor: t.colors.primary },
  primaryPressed: { backgroundColor: t.colors.primaryPressed },
  secondary: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.borderStrong },
  ghost: { backgroundColor: "transparent" },
  danger: { backgroundColor: t.colors.dangerSoft },
  pressed: { opacity: 0.75 },
  disabled: { opacity: 0.5 },
  fullWidth: { alignSelf: "stretch" },
  content: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm }
}));
