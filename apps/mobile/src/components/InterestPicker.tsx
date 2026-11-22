import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { Chip, Text, spacing } from "../ui";

const categoryLabels: Record<string, string> = {
  outdoors: "Plein air",
  sport: "Sport",
  food: "Table",
  culture: "Culture",
  creative: "Création",
  lifestyle: "Art de vivre"
};

type Props = {
  catalog: Interest[];
  selected: string[];
  onChange: (slugs: string[]) => void;
  max?: number;
};

export function InterestPicker({ catalog, selected, onChange, max = 10 }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, Interest[]>();
    for (const i of catalog) {
      map.set(i.category, [...(map.get(i.category) ?? []), i]);
    }
    return [...map.entries()];
  }, [catalog]);

  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < max) {
      onChange([...selected, slug]);
    }
  };

  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="caption">
        {selected.length}/{max} sélectionnés
      </Text>
      {groups.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <Text tone="subtle" variant="eyebrow">
            {categoryLabels[category] ?? category}
          </Text>
          <View style={styles.chips}>
            {items.map((i) => (
              <Chip
                key={i.slug}
                label={i.label}
                onPress={() => toggle(i.slug)}
                selected={selected.includes(i.slug)}
                testID={`interest-${i.slug}`}
              />
            ))}
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.md
  },
  group: {
    gap: spacing.xs
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
