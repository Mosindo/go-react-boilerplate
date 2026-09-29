import React from "react";
import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { controls } from "./tokens";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type IconButtonProps = Omit<PressableProps, "children" | "style" | "accessibilityLabel"> & {
  /** A short text glyph such as "×", "‹", "↑". */
  glyph: string;
  /** Required: icon-only controls must be announced. */
  accessibilityLabel: string;
  tone?: "default" | "primary" | "danger";
  filled?: boolean;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  base: {
    width: controls.minTarget,
    height: controls.minTarget,
    borderRadius: controls.minTarget / 2,
    alignItems: "center" as const,
    justifyContent: "center" as const
  },
  filled: { backgroundColor: t.colors.surfaceMuted },
  disabled: { opacity: 0.4 }
});

export function IconButton({
  accessibilityLabel,
  disabled,
  filled = false,
  glyph,
  style,
  tone = "default",
  ...props
}: IconButtonProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <Pressable
      accessibilityLabel={accessibilityLabel}
      accessibilityRole="button"
      accessibilityState={{ disabled: Boolean(disabled) }}
      disabled={disabled}
      hitSlop={4}
      style={[styles.base, filled ? styles.filled : null, disabled ? styles.disabled : null, style]}
      {...props}
    >
      <Text importantForAccessibility="no" tone={tone} variant="heading" weight="bold">
        {glyph}
      </Text>
    </Pressable>
  );
}
