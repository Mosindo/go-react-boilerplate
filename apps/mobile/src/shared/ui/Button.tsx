import React, { useRef } from "react";
import { ActivityIndicator, Animated, Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { MIN_TOUCH, radius, spacing, useColors } from "./theme";

type Variant = "primary" | "accent" | "secondary" | "ghost" | "danger";

type Props = {
  title: string;
  onPress: () => void;
  variant?: Variant;
  loading?: boolean;
  disabled?: boolean;
  testID?: string;
  accessibilityLabel?: string;
  style?: StyleProp<ViewStyle>;
};

export function Button({ title, onPress, variant = "primary", loading = false, disabled = false, testID, accessibilityLabel, style }: Props) {
  const c = useColors();
  const scale = useRef(new Animated.Value(1)).current;
  const inactive = disabled || loading;

  const palette: Record<Variant, { bg: string; fg: string; border: string }> = {
    primary: { bg: c.primary, fg: c.onPrimary, border: c.primary },
    accent: { bg: c.accent, fg: c.onAccent, border: c.accent },
    secondary: { bg: "transparent", fg: c.text, border: c.border },
    ghost: { bg: "transparent", fg: c.primary, border: "transparent" },
    danger: { bg: "transparent", fg: c.danger, border: c.danger },
  };
  const p = palette[variant];

  const animate = (to: number) => Animated.spring(scale, { toValue: to, useNativeDriver: true, speed: 40, bounciness: 0 }).start();

  return (
    <Animated.View style={[{ transform: [{ scale }] }, style]}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={accessibilityLabel ?? title}
        accessibilityState={{ disabled: inactive, busy: loading }}
        disabled={inactive}
        onPress={onPress}
        onPressIn={() => animate(0.97)}
        onPressOut={() => animate(1)}
        testID={testID}
        style={[styles.base, { backgroundColor: p.bg, borderColor: p.border, opacity: inactive ? 0.55 : 1 }]}
      >
        {loading ? <ActivityIndicator color={p.fg} /> : <Text variant="label" style={{ color: p.fg, fontSize: 16 }}>{title}</Text>}
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: MIN_TOUCH + 6,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center",
  },
});
