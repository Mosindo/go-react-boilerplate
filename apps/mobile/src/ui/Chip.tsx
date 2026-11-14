import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, type Palette } from "../theme/palette";
import { useStyles, useTheme } from "../theme/ThemeProvider";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  icon?: IconName;
  disabled?: boolean;
  size?: "sm" | "md";
};

export function Chip({ label, selected, onPress, icon, disabled, size = "md" }: Props) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const fg = selected ? colors.onPrimary : colors.text;
  const content = (
    <>
      {icon ? <Icon color={fg} name={icon} size={15} /> : null}
      <Text style={{ color: fg }} variant={size === "sm" ? "caption" : "label"} weight={size === "sm" ? "600" : undefined}>
        {label}
      </Text>
    </>
  );
  if (!onPress) {
    return <Pressable style={[styles.base, styles[size], selected ? styles.selected : null]}>{content}</Pressable>;
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected, disabled: !!disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [styles.base, styles[size], selected ? styles.selected : null, pressed ? styles.pressed : null, disabled && !selected ? styles.disabled : null]}
    >
      {content}
    </Pressable>
  );
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    base: {
      flexDirection: "row",
      alignItems: "center",
      gap: 6,
      borderRadius: radii.pill,
      borderWidth: 1,
      borderColor: c.border,
      backgroundColor: c.surface
    },
    md: { paddingHorizontal: spacing.md, minHeight: 40 },
    sm: { paddingHorizontal: spacing.sm, minHeight: 30 },
    selected: { backgroundColor: c.primary, borderColor: c.primary },
    pressed: { opacity: 0.8 },
    disabled: { opacity: 0.45 }
  });
