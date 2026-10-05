import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

/** Pill used for interests and options. Pass `onPress` to make it selectable. */
export function Chip({
  label,
  selected,
  onPress,
  disabled
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
}) {
  const { colors } = useTheme();
  const content = (
    <Text variant="label" style={{ color: selected ? colors.onPrimary : colors.text }}>
      {label}
    </Text>
  );
  const base = [
    styles.chip,
    { backgroundColor: selected ? colors.primary : colors.surface, borderColor: selected ? colors.primary : colors.border }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={base}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: !!selected, disabled: !!disabled }}
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [...base, { opacity: disabled ? 0.5 : pressed ? 0.8 : 1 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    minHeight: 36,
    justifyContent: "center"
  }
});
