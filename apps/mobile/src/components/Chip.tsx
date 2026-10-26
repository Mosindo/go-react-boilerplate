import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radius, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string };

export function Chip({ label, selected, onPress, testID }: Props) {
  const theme = useTheme();
  const interactive = !!onPress;
  return (
    <Pressable
      testID={testID}
      accessibilityRole={interactive ? "button" : "text"}
      accessibilityState={interactive ? { selected: !!selected } : undefined}
      accessibilityLabel={label}
      disabled={!interactive}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? theme.primarySoft : theme.surface,
          borderColor: selected ? theme.primary : theme.border
        }
      ]}
    >
      <Text variant="label" color={selected ? theme.primary : theme.text}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1.5,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.md + 2,
    paddingVertical: spacing.sm,
    minHeight: 36,
    justifyContent: "center"
  }
});
