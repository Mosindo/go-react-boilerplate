import React from "react";
import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { controls } from "./tokens";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type ChipProps = Omit<PressableProps, "children" | "style"> & {
  label: string;
  selected?: boolean;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  base: {
    minHeight: controls.minTarget,
    paddingHorizontal: t.spacing.lg,
    borderRadius: t.radii.pill,
    borderWidth: 1,
    borderColor: t.colors.borderStrong,
    backgroundColor: t.colors.surface,
    alignItems: "center" as const,
    justifyContent: "center" as const
  },
  selected: { backgroundColor: t.colors.primarySoft, borderColor: t.colors.primary, borderWidth: 2 },
  disabled: { opacity: 0.5 }
});

/** Toggle chip (multi-select or single-select option). */
export function Chip({ accessibilityLabel, disabled, label, selected = false, style, ...props }: ChipProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled: Boolean(disabled) }}
      disabled={disabled}
      style={[styles.base, selected ? styles.selected : null, disabled ? styles.disabled : null, style]}
      {...props}
    >
      <Text tone={selected ? "primary" : "default"} variant="label" weight={selected ? "bold" : "medium"}>
        {label}
      </Text>
    </Pressable>
  );
}
