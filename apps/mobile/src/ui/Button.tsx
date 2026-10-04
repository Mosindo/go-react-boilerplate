import React, { useRef, type ReactNode } from "react";
import {
  ActivityIndicator,
  Animated,
  Pressable,
  StyleSheet,
  View,
  type PressableProps,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { MIN_TOUCH, radii, spacing, useColors } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: ButtonVariant;
  loading?: boolean;
  icon?: IconName;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
};

/** Press feedback shared by buttons: a subtle scale-down. */
export function usePressScale(to = 0.97) {
  const scale = useRef(new Animated.Value(1)).current;
  const animate = (value: number) =>
    Animated.spring(scale, { toValue: value, useNativeDriver: true, speed: 40, bounciness: 0 }).start();
  return {
    scale,
    onPressIn: () => animate(to),
    onPressOut: () => animate(1)
  };
}

export function Button({
  label,
  variant = "primary",
  loading = false,
  icon,
  fullWidth = true,
  disabled,
  style,
  onPressIn,
  onPressOut,
  ...rest
}: ButtonProps) {
  const colors = useColors();
  const press = usePressScale();
  const inactive = disabled || loading;

  const palette = {
    primary: { bg: colors.primary, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primarySoft },
    ghost: { bg: "transparent", fg: colors.text, border: colors.border },
    danger: { bg: colors.danger, fg: colors.onPrimary, border: colors.danger }
  }[variant];

  let content: ReactNode = (
    <>
      {icon ? <Icon color={palette.fg} name={icon} size={20} /> : null}
      <Text style={{ color: palette.fg }} weight="semibold">
        {label}
      </Text>
    </>
  );
  if (loading) content = <ActivityIndicator color={palette.fg} />;

  return (
    <Animated.View style={[fullWidth ? styles.full : styles.auto, { transform: [{ scale: press.scale }] }, style]}>
      <Pressable
        accessibilityLabel={label}
        accessibilityRole="button"
        accessibilityState={{ disabled: !!inactive, busy: loading }}
        disabled={inactive}
        {...rest}
        onPressIn={(event) => {
          press.onPressIn();
          onPressIn?.(event);
        }}
        onPressOut={(event) => {
          press.onPressOut();
          onPressOut?.(event);
        }}
        style={[styles.base, { backgroundColor: palette.bg, borderColor: palette.border }, inactive && styles.disabled]}
      >
        <View style={styles.row}>{content}</View>
      </Pressable>
    </Animated.View>
  );
}

type IconButtonProps = Omit<PressableProps, "style" | "children"> & {
  icon: IconName;
  label: string;
  size?: number;
  color?: string;
  filled?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, label, size = 24, color, filled, style, ...rest }: IconButtonProps) {
  const colors = useColors();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      hitSlop={8}
      {...rest}
      style={({ pressed }) => [
        styles.iconButton,
        filled && { backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border },
        pressed && { opacity: 0.6 },
        style
      ]}
    >
      <Icon color={color ?? colors.text} name={icon} size={size} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  full: { alignSelf: "stretch" },
  auto: { alignSelf: "flex-start" },
  base: {
    minHeight: MIN_TOUCH + 6,
    minWidth: MIN_TOUCH,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  disabled: { opacity: 0.5 },
  iconButton: {
    width: MIN_TOUCH,
    height: MIN_TOUCH,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center"
  }
});
