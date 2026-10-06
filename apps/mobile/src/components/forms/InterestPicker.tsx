import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterestCatalog } from "../../hooks/queries";
import { LoadingView } from "../../shared/feedback";
import { Badge, Text, spacing } from "../../shared/ui";

export const MAX_INTERESTS = 10;

type InterestPickerProps = {
  selectedIds: number[];
  onChange: (ids: number[]) => void;
};

export function InterestPicker({ selectedIds, onChange }: InterestPickerProps) {
  const catalog = useInterestCatalog();
  if (catalog.isLoading) {
    return <LoadingView label="Chargement…" />;
  }
  const toggle = (id: number) => {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((existing) => existing !== id));
    } else if (selectedIds.length < MAX_INTERESTS) {
      onChange([...selectedIds, id]);
    }
  };
  return (
    <View style={styles.root}>
      <View style={styles.chips}>
        {(catalog.data ?? []).map((interest) => (
          <Badge
            key={interest.id}
            label={interest.label}
            onPress={() => toggle(interest.id)}
            selected={selectedIds.includes(interest.id)}
          />
        ))}
      </View>
      <Text tone="muted" variant="caption">
        {selectedIds.length}/{MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
