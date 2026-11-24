import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../../api/types";
import { colors, spacing } from "../../theme/tokens";
import { AppText } from "../AppText";
import { Chip } from "../Chip";

const categoryLabels: Record<string, string> = {
  outdoor: "Plein air",
  sport: "Sport",
  lifestyle: "Art de vivre",
  culture: "Culture",
  music: "Musique",
  games: "Jeux",
  mind: "Curiosité"
};

export const MAX_INTERESTS = 10;

type Props = { catalog: Interest[]; selected: number[]; onChange: (ids: number[]) => void };

export function InterestsPicker({ catalog, selected, onChange }: Props) {
  const grouped = useMemo(() => {
    const map = new Map<string, Interest[]>();
    for (const interest of catalog) {
      map.set(interest.category, [...(map.get(interest.category) ?? []), interest]);
    }
    return [...map.entries()];
  }, [catalog]);

  const toggle = (id: number) => {
    if (selected.includes(id)) onChange(selected.filter((x) => x !== id));
    else if (selected.length < MAX_INTERESTS) onChange([...selected, id]);
  };

  return (
    <View style={styles.wrap}>
      <AppText variant="caption" color={colors.textMuted}>
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </AppText>
      {grouped.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <AppText variant="overline" color={colors.textSubtle}>
            {categoryLabels[category] ?? category}
          </AppText>
          <View style={styles.chips}>
            {items.map((i) => (
              <Chip key={i.id} label={i.label} selected={selected.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
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
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
