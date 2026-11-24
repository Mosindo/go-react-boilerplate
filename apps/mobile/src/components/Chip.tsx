import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { colors, radii, spacing } from "../theme/tokens";
import { AppText } from "./AppText";

type Props = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, testID }: Props) {
  const content = (
    <AppText variant="label" color={selected ? colors.onPrimary : highlighted ? colors.accent : colors.text}>
      {label}
    </AppText>
  );
  const style = [styles.chip, selected && styles.selected, highlighted && !selected && styles.highlighted];
  if (!onPress) {
    return (
      <Pressable disabled style={style} accessibilityLabel={highlighted ? `${label}, en commun` : label}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [...style, pressed && styles.pressed]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surfaceRaised,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 1
  },
  selected: { backgroundColor: colors.primary, borderColor: colors.primary },
  highlighted: { borderColor: colors.accent, backgroundColor: colors.accentSoft },
  pressed: { opacity: 0.8 }
});
