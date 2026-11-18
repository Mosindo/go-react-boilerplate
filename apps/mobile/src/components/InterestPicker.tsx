import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { spacing } from "../theme";
import { Chip, LoadingState, Text } from "../ui";

export const MAX_INTERESTS = 10;

type Props = {
  catalog: Interest[] | undefined;
  selected: string[];
  onChange: (slugs: string[]) => void;
};

export function InterestPicker({ catalog, selected, onChange }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, Interest[]>();
    for (const interest of catalog ?? []) {
      map.set(interest.category, [...(map.get(interest.category) ?? []), interest]);
    }
    return [...map.entries()];
  }, [catalog]);

  if (!catalog) {
    return <LoadingState />;
  }
  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, slug]);
    }
  };
  return (
    <View style={styles.wrap}>
      <Text color="textMuted" variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      {groups.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <Text color="textSubtle" variant="overline">
            {category}
          </Text>
          <View style={styles.row}>
            {items.map((interest) => (
              <Chip
                key={interest.slug}
                label={interest.label}
                onPress={() => toggle(interest.slug)}
                selected={selected.includes(interest.slug)}
                testID={`interest-${interest.slug}`}
              />
            ))}
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  group: { gap: spacing.xs },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
