import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, useTheme } from "../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  highlight?: boolean;
};

export function Chip({ label, selected, onPress, highlight }: Props) {
  const t = useTheme();
  const active = selected || highlight;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={onPress ? { selected: !!selected } : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: active ? t.primarySoft : t.surface,
          borderColor: active ? t.primary : t.border
        }
      ]}
    >
      <Text variant="caption" style={{ color: active ? t.primary : t.text, fontWeight: "600" }}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 8
  }
});
