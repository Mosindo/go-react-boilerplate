import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

export type ChipProps = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, testID }: ChipProps) {
  const content = (
    <Text style={{ color: selected ? colors.textOnAccent : highlighted ? colors.accentStrong : colors.text }} variant="caption">
      {label}
    </Text>
  );
  const style = [styles.base, selected ? styles.selected : null, highlighted && !selected ? styles.highlighted : null];
  if (!onPress) {
    return (
      <Pressable accessible={false} disabled style={style} testID={testID}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      onPress={onPress}
      style={({ pressed }) => [...style, pressed ? styles.pressed : null]}
      testID={testID}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 7,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surfaceRaised
  },
  selected: {
    backgroundColor: colors.accent,
    borderColor: colors.accent
  },
  highlighted: {
    borderColor: colors.accent,
    backgroundColor: colors.accentSoft
  },
  pressed: {
    opacity: 0.8
  }
});
