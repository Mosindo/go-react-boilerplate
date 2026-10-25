import React from "react";
import {
  ActivityIndicator,
  Animated,
  Pressable,
  StyleSheet,
  type StyleProp,
  type ViewStyle,
  useAnimatedValue
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { radius, spacing, useTheme } from "../../theme";
import type { IconName } from "./IconButton";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type Props = {
  label: string;
  onPress: () => void;
  variant?: ButtonVariant;
  icon?: IconName;
  loading?: boolean;
  disabled?: boolean;
  accessibilityHint?: string;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function Button({
  label,
  onPress,
  variant = "primary",
  icon,
  loading = false,
  disabled = false,
  accessibilityHint,
  testID,
  style
}: Props) {
  const { colors } = useTheme();
  const scale = useAnimatedValue(1);
  const inactive = disabled || loading;

  const palette = {
    primary: { background: colors.primary, foreground: colors.onPrimary, border: colors.primary },
    secondary: { background: colors.surface, foreground: colors.text, border: colors.border },
    ghost: { background: "transparent", foreground: colors.primary, border: "transparent" },
    danger: { background: colors.danger, foreground: colors.onDanger, border: colors.danger }
  }[variant];

  const animateTo = (value: number) => {
    Animated.spring(scale, { toValue: value, useNativeDriver: true, speed: 40, bounciness: 0 }).start();
  };

  return (
    <Pressable
      accessibilityHint={accessibilityHint}
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      onPressIn={() => animateTo(0.97)}
      onPressOut={() => animateTo(1)}
      testID={testID}
    >
      <Animated.View
        style={[
          styles.base,
          {
            backgroundColor: palette.background,
            borderColor: palette.border,
            opacity: disabled ? 0.5 : 1,
            transform: [{ scale }]
          },
          style
        ]}
      >
        {loading ? (
          <ActivityIndicator color={palette.foreground} />
        ) : (
          <>
            {icon ? <Ionicons color={palette.foreground} name={icon} size={20} /> : null}
            <Text style={{ color: palette.foreground }} variant="label">
              {label}
            </Text>
          </>
        )}
      </Animated.View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 50,
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm
  }
});
