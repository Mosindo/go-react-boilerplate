import React from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { spacing } from "../theme";
import { Chip, Text } from "../shared/ui";

export const MAX_INTERESTS = 10;

export type InterestPickerProps = {
  interests: Interest[];
  selectedIds: number[];
  onChange: (ids: number[]) => void;
};

export function InterestPicker({ interests, selectedIds, onChange }: InterestPickerProps) {
  const full = selectedIds.length >= MAX_INTERESTS;
  const toggle = (id: number) => {
    onChange(selectedIds.includes(id) ? selectedIds.filter((item) => item !== id) : [...selectedIds, id]);
  };
  return (
    <View style={styles.wrap}>
      <Text accessibilityLiveRegion="polite" tone="muted" variant="caption">
        {selectedIds.length} of {MAX_INTERESTS} selected
      </Text>
      <View style={styles.chips}>
        {interests.map((interest) => {
          const selected = selectedIds.includes(interest.id);
          return (
            <Chip
              disabled={full && !selected}
              key={interest.id}
              label={interest.label}
              onPress={() => toggle(interest.id)}
              selected={selected}
            />
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
