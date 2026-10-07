import React from "react";
import { StyleSheet, View } from "react-native";
import { Text, radii, spacing } from "../../shared/ui";
import { usePalette } from "./palette";

type Props = { label: string; highlighted?: boolean; onDark?: boolean };

export function Chip({ label, highlighted = false, onDark = false }: Props) {
  const p = usePalette();
  const background = highlighted ? p.primary : onDark ? "rgba(255,255,255,0.22)" : p.surfaceMuted;
  const color = highlighted ? p.onPrimary : onDark ? "#ffffff" : p.text;
  return (
    <View
      accessibilityLabel={highlighted ? `${label}, shared interest` : label}
      style={[styles.chip, { backgroundColor: background }]}
    >
      <Text style={{ color }} variant="caption" weight={highlighted ? "bold" : "medium"}>
        {label}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  chip: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xxs + 1, borderRadius: radii.pill }
});
