import React from "react";
import { View } from "react-native";
import { makeStyles } from "../../theme";
import { Chip, LoadingState, Text } from "../../ui";
import { useInterestsCatalogue } from "./hooks";

const MAX = 10;

type Props = { selected: number[]; onChange: (ids: number[]) => void };

export function InterestPicker({ selected, onChange }: Props) {
  const styles = useStyles();
  const catalogue = useInterestsCatalogue();
  if (catalogue.isLoading) {
    return <LoadingState />;
  }
  const toggle = (id: number) => {
    if (selected.includes(id)) {
      onChange(selected.filter((x) => x !== id));
    } else if (selected.length < MAX) {
      onChange([...selected, id]);
    }
  };
  return (
    <View style={styles.wrapper}>
      <Text variant="caption" tone="subtle">
        {selected.length}/{MAX} sélectionnés
      </Text>
      <View style={styles.chips}>
        {(catalogue.data ?? []).map((interest) => (
          <Chip
            key={interest.id}
            testID={`interest-${interest.slug}`}
            label={interest.label}
            selected={selected.includes(interest.id)}
            disabled={!selected.includes(interest.id) && selected.length >= MAX}
            onPress={() => toggle(interest.id)}
          />
        ))}
      </View>
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  wrapper: { gap: t.spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm }
}));
