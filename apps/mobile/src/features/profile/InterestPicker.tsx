import React from "react";
import { StyleSheet, View } from "react-native";
import { Chip, LoadingState, Text } from "../../design/components";
import { useInterests } from "./hooks";

export const MAX_INTERESTS = 10;

type Props = { selected: number[]; onChange: (ids: number[]) => void };

export function InterestPicker({ selected, onChange }: Props) {
  const { data, isLoading } = useInterests();
  if (isLoading || !data) {
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
      <Text tone="muted" variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <View style={styles.chips}>
        {data.map((interest) => (
          <Chip
            key={interest.id}
            label={interest.label}
            onPress={() => toggle(interest.id)}
            selected={selected.includes(interest.id)}
            testID={`interest-${interest.slug}`}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 10 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
