import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../../hooks/queries";
import { Chip, Loader, Text, spacing } from "../../shared/ui";

export const MAX_INTERESTS = 10;

type InterestPickerProps = {
  value: number[];
  onChange: (value: number[]) => void;
};

export function InterestPicker({ value, onChange }: InterestPickerProps) {
  const { data, isLoading, isError } = useInterests();
  if (isLoading) {
    return <Loader />;
  }
  if (isError || !data) {
    return <Text tone="danger">Impossible de charger les centres d'intérêt.</Text>;
  }
  const toggle = (id: number) => {
    if (value.includes(id)) {
      onChange(value.filter((v) => v !== id));
    } else if (value.length < MAX_INTERESTS) {
      onChange([...value, id]);
    }
  };
  return (
    <View style={styles.wrap}>
      {data.map((interest) => (
        <Chip
          disabled={!value.includes(interest.id) && value.length >= MAX_INTERESTS}
          key={interest.id}
          label={interest.label}
          onPress={() => toggle(interest.id)}
          selected={value.includes(interest.id)}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm } });
