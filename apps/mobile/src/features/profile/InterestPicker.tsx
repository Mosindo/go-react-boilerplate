import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../../hooks/queries";
import { Chip, LoadingState, Text } from "../../ui";

export const MAX_INTERESTS = 8;

export function InterestPicker({ value, onChange }: { value: number[]; onChange: (ids: number[]) => void }) {
  const { data, isLoading } = useInterests();
  if (isLoading || !data) return <LoadingState />;
  const toggle = (id: number) => onChange(value.includes(id) ? value.filter((v) => v !== id) : [...value, id]);
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="caption">
        {value.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <View style={styles.chips}>
        {data.map((interest) => {
          const selected = value.includes(interest.id);
          return (
            <Chip
              disabled={!selected && value.length >= MAX_INTERESTS}
              key={interest.id}
              label={interest.label}
              onPress={() => toggle(interest.id)}
              selected={selected}
              testID={`interest-${interest.slug}`}
            />
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 10 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
