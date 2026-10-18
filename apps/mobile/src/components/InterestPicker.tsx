import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../hooks/queries";
import { Loader, Text, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { MAX_INTERESTS } from "./labels";

type InterestPickerProps = {
  selected: string[];
  onChange: (slugs: string[]) => void;
};

export function InterestPicker({ selected, onChange }: InterestPickerProps) {
  const { data, isLoading, isError } = useInterests();

  if (isLoading) {
    return <Loader label="Chargement des centres d'intérêt..." />;
  }
  if (isError || !data) {
    return <Text tone="danger">Impossible de charger les centres d'intérêt.</Text>;
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
      <Text tone="muted" variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <ChipRow>
        {data.map((interest) => {
          const isSelected = selected.includes(interest.slug);
          return (
            <Chip
              disabled={!isSelected && selected.length >= MAX_INTERESTS}
              key={interest.slug}
              label={interest.label}
              onPress={() => toggle(interest.slug)}
              selected={isSelected}
              testID={`interest-${interest.slug}`}
            />
          );
        })}
      </ChipRow>
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.sm } });
