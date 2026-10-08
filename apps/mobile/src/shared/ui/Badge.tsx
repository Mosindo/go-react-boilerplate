import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

/** Small count bubble, used on tab icons. */
export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={`${count} non lus`} style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text style={{ color: colors.onPrimary }} variant="caption" weight="bold">
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

type ChipProps = { label: string; selected?: boolean; onPress?: () => void; highlighted?: boolean; disabled?: boolean };

/** Pill used for interests and choices; selectable when `onPress` is provided. */
export function Chip({ label, selected = false, onPress, highlighted = false, disabled = false }: ChipProps) {
  const { colors } = useTheme();
  const active = selected || highlighted;
  const body = (
    <View
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primary : highlighted ? colors.primarySoft : colors.surfaceMuted,
          borderColor: active ? colors.primary : colors.border
        }
      ]}
    >
      <Text style={{ color: selected ? colors.onPrimary : highlighted ? colors.primary : colors.text }} variant="bodySmall" weight="medium">
        {label}
      </Text>
    </View>
  );
  if (!onPress) return body;
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={disabled ? styles.disabled : undefined}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  badge: { minWidth: 18, height: 18, borderRadius: 9, paddingHorizontal: 5, alignItems: "center", justifyContent: "center" },
  chip: { borderRadius: radii.pill, borderWidth: 1, paddingHorizontal: spacing.md, paddingVertical: spacing.xs + 2 },
  disabled: { opacity: 0.45 }
});
