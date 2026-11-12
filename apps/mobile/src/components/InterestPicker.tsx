import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { spacing } from "../theme";
import { Chip, Text } from "../ui";

const categoryLabels: Record<string, string> = { lifestyle: "Art de vivre", sport: "Sport", culture: "Culture", leisure: "Loisirs" };
export const MAX_INTERESTS = 10;

export function InterestPicker({ catalogue, selected, onToggle }: { catalogue: Interest[]; selected: number[]; onToggle: (id: number) => void }) {
  const groups = useMemo(() => {
    const map = new Map<string, Interest[]>();
    for (const i of catalogue) map.set(i.category, [...(map.get(i.category) ?? []), i]);
    return [...map.entries()];
  }, [catalogue]);
  const full = selected.length >= MAX_INTERESTS;
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="muted">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      {groups.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <Text variant="overline" tone="subtle">
            {categoryLabels[category] ?? category}
          </Text>
          <View style={styles.chips}>
            {items.map((i) => {
              const isOn = selected.includes(i.id);
              return <Chip key={i.id} label={i.label} selected={isOn} disabled={full && !isOn} onPress={() => onToggle(i.id)} />;
            })}
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
