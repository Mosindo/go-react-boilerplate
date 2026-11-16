import React from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { spacing } from "../theme/tokens";
import { Chip, LoadingState, Text } from "../ui";

export const MAX_INTERESTS = 10;

type Props = {
  interests: Interest[] | undefined;
  selected: number[];
  onChange: (ids: number[]) => void;
};

export function InterestPicker({ interests, selected, onChange }: Props) {
  if (!interests) {
    return <LoadingState />;
  }
  const toggle = (id: number) => {
    if (selected.includes(id)) {
      onChange(selected.filter((x) => x !== id));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, id]);
    }
  };
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="muted">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <View style={styles.chips}>
        {interests.map((i) => (
          <Chip key={i.id} label={i.label} selected={selected.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
