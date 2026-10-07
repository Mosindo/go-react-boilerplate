import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { Text, radii, spacing } from "../ui";

export type ChipOption<T extends string> = { value: T; label: string };

type ChipGroupProps<T extends string> = {
  options: ChipOption<T>[];
  selected: T[];
  onToggle: (value: T) => void;
  /** Disable unselected chips (e.g. when a max selection count is reached). */
  disableUnselected?: boolean;
};

export function ChipGroup<T extends string>({
  disableUnselected = false,
  onToggle,
  options,
  selected
}: ChipGroupProps<T>) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      {options.map((option) => {
        const isSelected = selected.includes(option.value);
        const disabled = disableUnselected && !isSelected;
        return (
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ selected: isSelected, disabled }}
            disabled={disabled}
            key={option.value}
            onPress={() => onToggle(option.value)}
            style={[
              styles.chip,
              {
                backgroundColor: isSelected ? colors.primary : colors.backgroundElevated,
                borderColor: isSelected ? colors.primary : colors.borderStrong,
                opacity: disabled ? 0.45 : 1
              }
            ]}
          >
            <Text
              style={isSelected ? { color: colors.primaryForeground } : null}
              variant="label"
              weight="semibold"
            >
              {option.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    minHeight: 40,
    justifyContent: "center"
  }
});
