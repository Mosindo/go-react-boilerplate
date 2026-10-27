import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, usePalette } from "../theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string };

export function Chip({ label, selected = false, onPress, testID }: Props) {
  const p = usePalette();
  const content = (
    <Text variant="label" style={{ color: selected ? p.primaryText : p.text }}>
      {label}
    </Text>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: selected ? p.primary : p.surface,
      borderColor: selected ? p.primary : p.border
    }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={style}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      accessibilityLabel={label}
      onPress={onPress}
      style={style}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: radii.pill,
    borderWidth: 1
  }
});
