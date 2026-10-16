import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  tone?: "default" | "highlight";
  testID?: string;
};

export function Chip({ label, selected = false, onPress, disabled = false, tone = "default", testID }: ChipProps) {
  const body = (
    <View
      style={[
        styles.chip,
        selected ? styles.selected : null,
        tone === "highlight" ? styles.highlight : null,
        disabled ? styles.disabled : null
      ]}
    >
      <Text tone={selected ? "inverse" : "default"} variant="label" weight="semibold">
        {label}
      </Text>
    </View>
  );
  if (!onPress) {
    return body;
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected, disabled }}
      disabled={disabled}
      hitSlop={4}
      onPress={onPress}
      testID={testID}
    >
      {body}
    </Pressable>
  );
}

export function ChipRow({ children }: { children: React.ReactNode }) {
  return <View style={styles.row}>{children}</View>;
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated
  },
  selected: { backgroundColor: colors.primary, borderColor: colors.primary },
  highlight: { backgroundColor: colors.primarySoft, borderColor: colors.primaryBorder },
  disabled: { opacity: 0.5 }
});
